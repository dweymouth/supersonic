package browsing

import (
	"log"
	"time"

	"github.com/dweymouth/supersonic/backend"
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

	"github.com/dweymouth/go-subsonic/subsonic"
)

type FavoritesPage struct {
	widget.BaseWidget

	cfg   *backend.FavoritesPageConfig
	contr *controller.Controller
	pm    *backend.PlaybackManager
	im    *backend.ImageManager
	sm    *backend.ServerManager
	lm    *backend.LibraryManager

	searchText        string
	nowPlayingID      string
	pendingViewSwitch bool

	grid          *widgets.GridView
	searchGrid    *widgets.GridView
	artistListCtr *fyne.Container
	tracklistCtr  *fyne.Container
	searcher      *widgets.SearchEntry
	titleDisp     *widget.RichText
	toggleBtns    *widgets.ToggleButtonGroup
	container     *fyne.Container
}

func NewFavoritesPage(cfg *backend.FavoritesPageConfig, contr *controller.Controller, sm *backend.ServerManager, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager) *FavoritesPage {
	a := &FavoritesPage{
		cfg:   cfg,
		contr: contr,
		pm:    pm,
		lm:    lm,
		sm:    sm,
		im:    im,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(0, "")
	a.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(lm.StarredIter()), a.im)
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

func (a *FavoritesPage) createHeader(activeBtnIdx int, searchText string) {
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
	a.searcher.Entry.Text = searchText
}

func (a *FavoritesPage) createContainer(initialView fyne.CanvasObject) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(9), a.titleDisp, container.NewCenter(a.toggleBtns), layout.NewSpacer(), searchVbox, util.NewHSpace(15)),
		nil, nil, nil, initialView)
}

func restoreFavoritesPage(saved *savedFavoritesPage) *FavoritesPage {
	a := &FavoritesPage{
		cfg:   saved.cfg,
		contr: saved.contr,
		pm:    saved.pm,
		lm:    saved.lm,
		sm:    saved.sm,
		im:    saved.im,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(saved.activeToggleBtn, saved.searchText)
	a.grid = widgets.NewGridViewFromState(saved.gridState)

	if saved.searchText != "" {
		a.searchGrid = widgets.NewGridViewFromState(saved.searchGridState)
	}

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

func (a *FavoritesPage) Reload() {
	// reload favorite albums view
	if a.searchText != "" {
		a.doSearchAlbums(a.searchText)
	} else {
		a.grid.Reset(widgets.NewGridViewAlbumIterator(a.lm.StarredIter()))
	}
	if a.tracklistCtr != nil || a.artistListCtr != nil {
		go func() {
			// re-fetch starred info from server
			starred, err := a.sm.Server.GetStarred2(nil)
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			if a.tracklistCtr != nil {
				// refresh favorite songs view
				tr := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
				tr.Tracks = starred.Song
				if a.toggleBtns.ActivatedButtonIndex() == 2 {
					// favorite songs view is visible
					tr.Refresh()
				}
			}
			if a.artistListCtr != nil {
				// refresh favorite artists view
				al := a.artistListCtr.Objects[0].(*widgets.ArtistGenreList)
				al.Items = buildArtistListModel(starred.Artist)
				if a.toggleBtns.ActivatedButtonIndex() == 1 {
					// favorite artists view is visible
					al.Refresh()
				}
			}
		}()
	}
}

func (a *FavoritesPage) Save() SavedPage {
	sf := &savedFavoritesPage{
		cfg:             a.cfg,
		contr:           a.contr,
		pm:              a.pm,
		sm:              a.sm,
		im:              a.im,
		lm:              a.lm,
		searchText:      a.searchText,
		gridState:       a.grid.SaveToState(),
		activeToggleBtn: a.toggleBtns.ActivatedButtonIndex(),
	}
	if a.searchGrid != nil {
		sf.searchGridState = a.searchGrid.SaveToState()
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

func (a *FavoritesPage) OnSongChange(song *subsonic.Child, _ *subsonic.Child) {
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
	iter := a.lm.SearchIterWithFilter(query, func(al *subsonic.AlbumID3) bool {
		return al.Starred.After(time.Time{})
	})
	if a.searchGrid == nil {
		a.searchGrid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), a.im)
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
	if a.artistListCtr == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		if a.container == nil {
			a.createContainer(layout.NewSpacer())
		}
		go func() {
			s, err := a.sm.Server.GetStarred2(nil)
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			model := buildArtistListModel(s.Artist)
			artistList := widgets.NewArtistGenreList(model)
			artistList.ShowAlbumCount = true
			artistList.OnNavTo = func(artistID string) {
				a.contr.NavigateTo(controller.ArtistRoute(artistID))
			}
			a.artistListCtr = container.New(
				&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
				artistList)
			a.container.Objects[0] = a.artistListCtr
			a.Refresh()
			a.pendingViewSwitch = false
		}()
	} else {
		a.container.Objects[0] = a.artistListCtr
		a.Refresh()
	}
}

func buildArtistListModel(artists []*subsonic.ArtistID3) []widgets.ArtistGenreListItemModel {
	model := make([]widgets.ArtistGenreListItemModel, 0)
	for _, ar := range artists {
		model = append(model, widgets.ArtistGenreListItemModel{
			ID:         ar.ID,
			Name:       ar.Name,
			AlbumCount: ar.AlbumCount,
		})
	}
	return model
}

func (a *FavoritesPage) onShowFavoriteSongs() {
	a.cfg.InitialView = "Songs" // save setting
	a.searcher.Entry.Hide()     // disable search on songs for now
	if a.tracklistCtr == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		if a.container == nil {
			a.createContainer(layout.NewSpacer())
		}
		go func() {
			s, err := a.sm.Server.GetStarred2(nil)
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			tracklist := widgets.NewTracklist(s.Song)
			tracklist.AutoNumber = true
			tracklist.SetVisibleColumns(a.cfg.TracklistColumns)
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
	pm              *backend.PlaybackManager
	sm              *backend.ServerManager
	im              *backend.ImageManager
	lm              *backend.LibraryManager
	gridState       widgets.GridViewState
	searchGridState widgets.GridViewState
	searchText      string
	activeToggleBtn int
}

func (s *savedFavoritesPage) Restore() Page {
	return restoreFavoritesPage(s)
}
