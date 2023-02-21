package browsing

import (
	"log"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type FavoritesPage struct {
	widget.BaseWidget

	contr controller.Controller
	pm    *backend.PlaybackManager
	im    *backend.ImageManager
	sm    *backend.ServerManager
	lm    *backend.LibraryManager
	nav   func(Route)

	searchText        string
	nowPlayingID      string
	pendingViewSwitch bool

	grid          *widgets.AlbumGrid
	searchGrid    *widgets.AlbumGrid
	artistListCtr *fyne.Container
	tracklistCtr  *fyne.Container
	searcher      *widgets.Searcher
	titleDisp     *widget.RichText
	toggleBtns    *widgets.ToggleButtonGroup
	container     *fyne.Container
}

func NewFavoritesPage(contr controller.Controller, sm *backend.ServerManager, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *FavoritesPage {
	a := &FavoritesPage{
		contr: contr,
		pm:    pm,
		lm:    lm,
		sm:    sm,
		im:    im,
		nav:   nav,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(0, "")
	a.grid = widgets.NewAlbumGrid(a.lm.StarredIter(), a.im, false)
	a.connectGridActions()
	a.createContainer(false)
	return a
}

func (a *FavoritesPage) createHeader(activeBtnIdx int, searchText string) {
	a.titleDisp = widget.NewRichTextWithText("Favorites")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.toggleBtns = widgets.NewToggleButtonGroup(activeBtnIdx,
		widget.NewButtonWithIcon("", res.ResDiscInvertPng, a.onShowFavoriteAlbums),
		widget.NewButtonWithIcon("", res.ResPeopleInvertPng, a.onShowFavoriteArtists),
		widget.NewButtonWithIcon("", res.ResMusicnotesInvertPng, a.onShowFavoriteSongs))
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.OnSearched
	a.searcher.Entry.Text = searchText
}

func (a *FavoritesPage) connectGridActions() {
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.grid.OnShowAlbumPage = a.onShowAlbumPage
	a.grid.OnShowArtistPage = a.onShowArtistPage
}

func (a *FavoritesPage) createContainer(searchGrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher.Entry, layout.NewSpacer())
	gr := a.grid
	if searchGrid {
		gr = a.searchGrid
	}
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), a.titleDisp, container.NewCenter(a.toggleBtns), layout.NewSpacer(), searchVbox, widgets.NewHSpace(15)),
		nil, nil, nil, gr)
}

func restoreFavoritesPage(saved *savedFavoritesPage) *FavoritesPage {
	a := &FavoritesPage{
		contr: saved.contr,
		pm:    saved.pm,
		lm:    saved.lm,
		sm:    saved.sm,
		im:    saved.im,
		nav:   saved.nav,
	}
	a.ExtendBaseWidget(a)
	a.createHeader(saved.activeToggleBtn, saved.searchText)
	a.grid = widgets.NewAlbumGridFromState(saved.gridState)
	a.connectGridActions()

	if saved.searchText != "" {
		a.searchGrid = widgets.NewAlbumGridFromState(saved.searchGridState)
	}
	a.createContainer(saved.searchText != "")
	if saved.activeToggleBtn == 1 {
		a.onShowFavoriteArtists()
	} else if saved.activeToggleBtn == 2 {
		a.onShowFavoriteSongs()
	}

	return a
}

func (a *FavoritesPage) Route() Route {
	return FavoritesRoute()
}

func (a *FavoritesPage) Reload() {
	if a.searchText != "" {
		a.doSearchAlbums(a.searchText)
	} else {
		a.grid.Reset(a.lm.StarredIter())
	}
}

func (a *FavoritesPage) Save() SavedPage {
	sf := &savedFavoritesPage{
		contr:           a.contr,
		pm:              a.pm,
		sm:              a.sm,
		im:              a.im,
		lm:              a.lm,
		nav:             a.nav,
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
	return a.searcher.Entry
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

func (a *FavoritesPage) OnSongChange(song *subsonic.Child) {
	a.nowPlayingID = ""
	if song != nil {
		a.nowPlayingID = song.ID
	}
	if a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).SetNowPlaying(a.nowPlayingID)
	}
}

func (a *FavoritesPage) doSearchAlbums(query string) {
	iter := a.lm.SearchIterWithFilter(query, func(al *subsonic.AlbumID3) bool {
		return al.Starred.After(time.Time{})
	})
	if a.searchGrid == nil {
		a.searchGrid = widgets.NewAlbumGrid(iter, a.im, false /*showYear*/)
		a.searchGrid.OnPlayAlbum = a.onPlayAlbum
		a.searchGrid.OnShowAlbumPage = a.onShowAlbumPage
		a.searchGrid.OnShowArtistPage = a.onShowArtistPage
	} else {
		a.searchGrid.Reset(iter)
	}
	a.container.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *FavoritesPage) onShowFavoriteAlbums() {
	a.searcher.Entry.Show()
	if a.searchText == "" {
		a.container.Objects[0] = a.grid
	} else {
		a.container.Objects[0] = a.searchGrid
	}
	a.Refresh()
}

func (a *FavoritesPage) onShowFavoriteArtists() {
	a.searcher.Entry.Hide() // disable search on artists for now
	if a.artistListCtr == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		go func() {
			s, err := a.sm.Server.GetStarred2(nil)
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			model := make([]widgets.ArtistGenrePlaylistItemModel, 0)
			for _, ar := range s.Artist {
				model = append(model, widgets.ArtistGenrePlaylistItemModel{
					Name:       ar.Name,
					AlbumCount: ar.AlbumCount,
				})
			}
			artistList := widgets.NewArtistGenrePlaylist(model)
			artistList.ShowAlbumCount = true
			artistList.OnNavTo = func(artistID string) {
				a.nav(ArtistRoute(artistID))
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

func (a *FavoritesPage) onShowFavoriteSongs() {
	a.searcher.Entry.Hide() // disable search on songs for now
	if a.tracklistCtr == nil {
		if a.pendingViewSwitch {
			return
		}
		a.pendingViewSwitch = true
		go func() {
			s, err := a.sm.Server.GetStarred2(nil)
			if err != nil {
				log.Printf("error getting starred items: %s", err.Error())
				return
			}
			tracklist := widgets.NewTracklist(s.Song)
			tracklist.AutoNumber = true
			// TODO: get visible columns from config
			tracklist.SetVisibleColumns([]string{"Artist", "Album", "Plays"})
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

func (a *FavoritesPage) onPlayAlbum(albumID string) {
	go a.pm.PlayAlbum(albumID, 0)
}

func (a *FavoritesPage) onShowAlbumPage(albumID string) {
	a.nav(AlbumRoute(albumID))
}

func (a *FavoritesPage) onShowArtistPage(artistID string) {
	a.nav(ArtistRoute(artistID))
}

func (a *FavoritesPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedFavoritesPage struct {
	contr           controller.Controller
	pm              *backend.PlaybackManager
	sm              *backend.ServerManager
	im              *backend.ImageManager
	lm              *backend.LibraryManager
	gridState       widgets.AlbumGridState
	searchGridState widgets.AlbumGridState
	searchText      string
	activeToggleBtn int
	nav             func(Route)
}

func (s *savedFavoritesPage) Restore() Page {
	return restoreFavoritesPage(s)
}
