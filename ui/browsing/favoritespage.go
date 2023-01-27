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
	"github.com/dweymouth/go-subsonic"
)

type FavoritesPage struct {
	widget.BaseWidget

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

	OnPlayAlbum func(string, int)
}

func NewFavoritesPage(sm *backend.ServerManager, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *FavoritesPage {
	a := &FavoritesPage{
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
	a.createContainer()
	return a
}

func (a *FavoritesPage) createTitle() {
	a.titleDisp = widget.NewRichTextWithText("Favorites")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
}

func (a *FavoritesPage) createContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher.Entry, layout.NewSpacer())
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), a.titleDisp, layout.NewSpacer(), searchVbox, widgets.NewHSpace(15)),
		nil, nil, nil, a.grid)
}

func restoreFavoritesPage(saved *savedFavoritesPage) *FavoritesPage {
	a := &FavoritesPage{
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
	a.createContainer()

	return a
}

func (a *FavoritesPage) Route() Route {
	return FavoritesRoute()
}

func (a *FavoritesPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *FavoritesPage) Reload() {
	a.grid.Reset(a.lm.StarredIter())
}

func (a *FavoritesPage) Save() SavedPage {
	return &savedFavoritesPage{
		sm:        a.sm,
		im:        a.im,
		lm:        a.lm,
		nav:       a.nav,
		gridState: a.grid.SaveToState(),
	}
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
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID, 0)
	}
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
	sm        *backend.ServerManager
	im        *backend.ImageManager
	lm        *backend.LibraryManager
	gridState widgets.AlbumGridState
	nav       func(Route)
}

func (s *savedFavoritesPage) Restore() Page {
	return restoreFavoritesPage(s)
}
