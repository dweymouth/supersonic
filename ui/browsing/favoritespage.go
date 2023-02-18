package browsing

import (
	"supersonic/backend"
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

	pm         *backend.PlaybackManager
	im         *backend.ImageManager
	sm         *backend.ServerManager
	lm         *backend.LibraryManager
	nav        func(Route)
	grid       *widgets.AlbumGrid
	searchGrid *widgets.AlbumGrid
	searcher   *widgets.Searcher
	searchText string
	titleDisp  *widget.RichText
	container  *fyne.Container
}

func NewFavoritesPage(sm *backend.ServerManager, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *FavoritesPage {
	a := &FavoritesPage{
		pm:  pm,
		lm:  lm,
		sm:  sm,
		im:  im,
		nav: nav,
	}
	a.ExtendBaseWidget(a)
	a.createTitle()
	iter := lm.StarredIter()
	a.grid = widgets.NewAlbumGrid(iter, im, false)
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.grid.OnShowAlbumPage = a.onShowAlbumPage
	a.grid.OnShowArtistPage = a.onShowArtistPage
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.OnSearched
	a.createContainer(false)
	return a
}

func (a *FavoritesPage) createTitle() {
	a.titleDisp = widget.NewRichTextWithText("Favorites")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
}

func (a *FavoritesPage) createContainer(searchGrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher.Entry, layout.NewSpacer())
	gr := a.grid
	if searchGrid {
		gr = a.searchGrid
	}
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), a.titleDisp, layout.NewSpacer(), searchVbox, widgets.NewHSpace(15)),
		nil, nil, nil, gr)
}

func restoreFavoritesPage(saved *savedFavoritesPage) *FavoritesPage {
	a := &FavoritesPage{
		pm:  saved.pm,
		lm:  saved.lm,
		sm:  saved.sm,
		im:  saved.im,
		nav: saved.nav,
	}
	a.ExtendBaseWidget(a)
	a.createTitle()
	a.grid = widgets.NewAlbumGridFromState(saved.gridState)
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.grid.OnShowAlbumPage = a.onShowAlbumPage
	a.grid.OnShowArtistPage = a.onShowArtistPage
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.OnSearched
	a.searcher.Entry.Text = saved.searchText
	if saved.searchText != "" {
		a.searchGrid = widgets.NewAlbumGridFromState(saved.searchGridState)
	}
	a.createContainer(saved.searchText != "")

	return a
}

func (a *FavoritesPage) Route() Route {
	return FavoritesRoute()
}

func (a *FavoritesPage) Reload() {
	if a.searchText != "" {
		a.doSearch(a.searchText)
	} else {
		a.grid.Reset(a.lm.StarredIter())
	}
}

func (a *FavoritesPage) Save() SavedPage {
	sf := &savedFavoritesPage{
		pm:         a.pm,
		sm:         a.sm,
		im:         a.im,
		lm:         a.lm,
		nav:        a.nav,
		searchText: a.searchText,
		gridState:  a.grid.SaveToState(),
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
	a.doSearch(query)
}

func (a *FavoritesPage) doSearch(query string) {
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
	pm              *backend.PlaybackManager
	sm              *backend.ServerManager
	im              *backend.ImageManager
	lm              *backend.LibraryManager
	gridState       widgets.AlbumGridState
	searchGridState widgets.AlbumGridState
	searchText      string
	nav             func(Route)
}

func (s *savedFavoritesPage) Restore() Page {
	return restoreFavoritesPage(s)
}
