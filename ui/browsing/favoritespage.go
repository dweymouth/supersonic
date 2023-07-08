package browsing

import (
	"fmt"
	"log"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type FavoritesPage struct {
	widget.BaseWidget

	cfg   *backend.FavoritesPageConfig
	pool  *util.WidgetPool
	contr *controller.Controller
	pm    *backend.PlaybackManager
	im    *backend.ImageManager
	mp    mediaprovider.MediaProvider

	disposed          bool
	trackSort         widgets.TracklistSort
	filter            mediaprovider.AlbumFilter
	searchText        string
	nowPlayingID      string
	pendingViewSwitch bool

	grid         *widgets.GridView
	searchGrid   *widgets.GridView
	artistGrid   *widgets.GridView
	tracklistCtr *fyne.Container
	searcher     *widgets.SearchEntry
	filterBtn    *widgets.AlbumFilterButton
	titleDisp    *widget.RichText
	toggleBtns   *widgets.ToggleButtonGroup
	container    *fyne.Container
}

func NewFavoritesPage(cfg *backend.FavoritesPageConfig, pool *util.WidgetPool, contr *controller.Controller, mp mediaprovider.MediaProvider, pm *backend.PlaybackManager, im *backend.ImageManager) *FavoritesPage {
	a := &FavoritesPage{
		filter: mediaprovider.AlbumFilter{ExcludeUnfavorited: true},
		cfg:    cfg,
		pool:   pool,
		contr:  contr,
		pm:     pm,
		mp:     mp,
		im:     im,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(0)
	iter := mp.IterateAlbums("", a.filter)
	a.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), a.im, myTheme.AlbumIcon)
	a.contr.ConnectAlbumGridActions(a.grid)
	if cfg.InitialView == "Artists" {
		a.toggleBtns.SetActivatedButton(1)
		a.onShowFavoriteArtists()
	} else if cfg.InitialView == "Songs" {
		a.toggleBtns.SetActivatedButton(2)
		a.onShowFavoriteSongs()
	} else { // Albums view
		a.createContainer(a.grid)
	}
	return a
}

func (a *FavoritesPage) createHeader(activeBtnIdx int) {
	a.titleDisp = widget.NewRichTextWithText("Favorites")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.toggleBtns = widgets.NewToggleButtonGroup(activeBtnIdx,
		widget.NewButtonWithIcon("", myTheme.AlbumIcon, a.onShowFavoriteAlbums),
		widget.NewButtonWithIcon("", myTheme.ArtistIcon, a.onShowFavoriteArtists),
		widget.NewButtonWithIcon("", myTheme.TracksIcon, a.onShowFavoriteSongs))
	a.searcher = widgets.NewSearchEntry()
	a.searcher.OnSearched = a.OnSearched
	a.searcher.Entry.Text = a.searchText
	a.filterBtn = widgets.NewAlbumFilterButton(&a.filter, a.mp.GetGenres)
	a.filterBtn.FavoriteDisabled = true
	a.filterBtn.OnChanged = a.Reload
}

func (a *FavoritesPage) createContainer(initialView fyne.CanvasObject) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.NewBorder(container.NewHBox(util.NewHSpace(9),
		a.titleDisp, container.NewCenter(a.toggleBtns), layout.NewSpacer(), container.NewCenter(a.filterBtn), searchVbox, util.NewHSpace(15)),
		nil, nil, nil, initialView)
}

func restoreFavoritesPage(saved *savedFavoritesPage) *FavoritesPage {
	a := &FavoritesPage{
		cfg:        saved.cfg,
		contr:      saved.contr,
		pool:       saved.pool,
		pm:         saved.pm,
		mp:         saved.mp,
		im:         saved.im,
		searchText: saved.searchText,
		filter:     saved.filter,
		trackSort:  saved.trackSort,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(saved.activeToggleBtn)
	a.grid = widgets.NewGridViewFromState(saved.gridState)

	if saved.searchText != "" {
		a.searchGrid = widgets.NewGridViewFromState(saved.searchGridState)
	}
	a.toggleBtns.SetActivatedButton(saved.activeToggleBtn)

	if saved.activeToggleBtn == 1 {
		a.onShowFavoriteArtists()
	} else if saved.activeToggleBtn == 2 {
		a.onShowFavoriteSongs()
	} else if saved.searchText != "" {
		a.createContainer(a.searchGrid)
	} else {
		a.createContainer(a.grid)
	}

	return a
}

func (a *FavoritesPage) Route() controller.Route {
	return controller.FavoritesRoute()
}

func (a *FavoritesPage) Tapped(*fyne.PointEvent) {
	if a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).UnselectAll()
	}
}

func (a *FavoritesPage) Reload() {
	// reload favorite albums view
	if a.searchText != "" {
		a.doSearchAlbums(a.searchText)
	} else {
		iter := a.mp.IterateAlbums("", a.filter)
		a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
	}
	if a.tracklistCtr != nil || a.artistGrid != nil {
		go func() {
			// re-fetch starred info from server
			starred, err := a.mp.GetFavorites()
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			if a.disposed {
				return
			}
			if a.tracklistCtr != nil {
				// refresh favorite songs view
				tr := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
				tr.SetTracks(starred.Tracks)
				if a.toggleBtns.ActivatedButtonIndex() == 2 {
					// favorite songs view is visible
					tr.Refresh()
				}
			}
			if a.artistGrid != nil {
				// refresh favorite artists view
				a.artistGrid.ResetFixed(buildArtistGridViewModel(starred.Artists))
				if a.toggleBtns.ActivatedButtonIndex() == 1 {
					// favorite artists view is visible
					a.artistGrid.Refresh()
				}
			}
		}()
	}
}

func (a *FavoritesPage) Save() SavedPage {
	a.disposed = true
	sf := &savedFavoritesPage{
		cfg:             a.cfg,
		contr:           a.contr,
		pool:            a.pool,
		pm:              a.pm,
		mp:              a.mp,
		im:              a.im,
		filter:          a.filter,
		searchText:      a.searchText,
		gridState:       a.grid.SaveToState(),
		activeToggleBtn: a.toggleBtns.ActivatedButtonIndex(),
	}
	if a.searchGrid != nil {
		sf.searchGridState = a.searchGrid.SaveToState()
	}
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		sf.trackSort = tl.Sorting()
		a.pool.Release(util.WidgetTypeTracklist, tl)
	}
	return sf
}

var _ Searchable = (*FavoritesPage)(nil)

func (a *FavoritesPage) SearchWidget() fyne.Focusable {
	return a.searcher
}

func (a *FavoritesPage) OnSearched(query string) {
	a.searchText = query
	if query == "" {
		a.container.Objects[0] = a.grid
		if a.searchGrid != nil {
			a.searchGrid.Clear()
		}
		a.Refresh()
		return
	}
	a.doSearchAlbums(query)
}

var _ CanShowNowPlaying = (*FavoritesPage)(nil)

func (a *FavoritesPage) OnSongChange(song, _ *mediaprovider.Track) {
	a.nowPlayingID = ""
	if song != nil {
		a.nowPlayingID = song.ID
	}
	if a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).SetNowPlaying(a.nowPlayingID)
	}
}

var _ CanSelectAll = (*FavoritesPage)(nil)

func (a *FavoritesPage) SelectAll() {
	if a.toggleBtns.ActivatedButtonIndex() == 2 /*songs*/ && a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).SelectAll()
	}
}

func (a *FavoritesPage) doSearchAlbums(query string) {
	iter := a.mp.SearchAlbums(query, a.filter)
	if a.searchGrid == nil {
		a.searchGrid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), a.im, myTheme.AlbumIcon)
		a.contr.ConnectAlbumGridActions(a.searchGrid)
	} else {
		a.searchGrid.Reset(widgets.NewGridViewAlbumIterator(iter))
	}
	a.container.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *FavoritesPage) onShowFavoriteAlbums() {
	a.cfg.InitialView = "Albums" // save setting
	a.searcher.Entry.Show()
	a.filterBtn.Show()
	if a.searchText == "" {
		a.container.Objects[0] = a.grid
	} else {
		a.container.Objects[0] = a.searchGrid
	}
	a.Refresh()
}

func (a *FavoritesPage) onShowFavoriteArtists() {
	a.cfg.InitialView = "Artists" // save setting
	a.searcher.Entry.Hide()       // disable search on artists for now
	a.filterBtn.Hide()
	if a.artistGrid == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		if a.container == nil {
			a.createContainer(layout.NewSpacer())
		}
		go func() {
			fav, err := a.mp.GetFavorites()
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			if a.disposed {
				return
			}
			model := buildArtistGridViewModel(fav.Artists)
			a.artistGrid = widgets.NewFixedGridView(model, a.im, myTheme.ArtistIcon)
			a.contr.ConnectArtistGridActions(a.artistGrid)
			a.container.Objects[0] = a.artistGrid
			a.Refresh()
			a.pendingViewSwitch = false
		}()
	} else {
		a.container.Objects[0] = a.artistGrid
		a.Refresh()
	}
}

func buildArtistGridViewModel(artists []*mediaprovider.Artist) []widgets.GridViewItemModel {
	model := make([]widgets.GridViewItemModel, 0)
	for _, ar := range artists {
		albums := "albums"
		if ar.AlbumCount == 1 {
			albums = "album"
		}
		model = append(model, widgets.GridViewItemModel{
			ID:         ar.ID,
			CoverArtID: ar.CoverArtID,
			Name:       ar.Name,
			Secondary:  fmt.Sprintf("%d %s", ar.AlbumCount, albums),
		})
	}
	return model
}

func (a *FavoritesPage) onShowFavoriteSongs() {
	a.cfg.InitialView = "Songs" // save setting
	a.searcher.Entry.Hide()     // disable search on songs for now
	a.filterBtn.Hide()
	if a.tracklistCtr == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		if a.container == nil {
			a.createContainer(layout.NewSpacer())
		}
		go func() {
			fav, err := a.mp.GetFavorites()
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			if a.disposed {
				return
			}
			var tracklist *widgets.Tracklist
			if tl := a.pool.Obtain(util.WidgetTypeTracklist); tl != nil {
				tracklist = tl.(*widgets.Tracklist)
				tracklist.Reset()
				tracklist.SetTracks(fav.Tracks)
			} else {
				tracklist = widgets.NewTracklist(fav.Tracks)
			}
			tracklist.Options = widgets.TracklistOptions{AutoNumber: true}
			tracklist.SetVisibleColumns(a.cfg.TracklistColumns)
			tracklist.SetSorting(a.trackSort)
			tracklist.OnVisibleColumnsChanged = func(cols []string) {
				a.cfg.TracklistColumns = cols
			}
			tracklist.SetNowPlaying(a.nowPlayingID)
			a.contr.ConnectTracklistActions(tracklist)
			a.tracklistCtr = container.New(
				&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
				tracklist)
			a.container.Objects[0] = a.tracklistCtr
			a.Refresh()
			a.pendingViewSwitch = false
		}()
	} else {
		a.container.Objects[0] = a.tracklistCtr
		a.Refresh()
	}
}

func (a *FavoritesPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedFavoritesPage struct {
	cfg             *backend.FavoritesPageConfig
	contr           *controller.Controller
	pool            *util.WidgetPool
	pm              *backend.PlaybackManager
	mp              mediaprovider.MediaProvider
	im              *backend.ImageManager
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
	filter          mediaprovider.AlbumFilter
	searchText      string
	activeToggleBtn int
	trackSort       widgets.TracklistSort
}

func (s *savedFavoritesPage) Restore() Page {
	return restoreFavoritesPage(s)
}
