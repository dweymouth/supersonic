package browsing

import (
	"supersonic/backend"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

// TODO: there is a lot of code duplication between this and albumspage. Refactor?
type GenrePage struct {
	widget.BaseWidget

	genre      string
	im         *backend.ImageManager
	lm         *backend.LibraryManager
	nav        func(Route)
	grid       *widgets.AlbumGrid
	searchGrid *widgets.AlbumGrid
	searcher   *widgets.Searcher
	searchText string
	titleDisp  *widget.RichText

	OnPlayAlbum func(string, int)

	container *fyne.Container
}

func NewGenrePage(genre string, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *GenrePage {
	g := &GenrePage{
		genre: genre,
		lm:    lm,
		im:    im,
		nav:   nav,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	iter := g.lm.GenreIter(g.genre)
	g.grid = widgets.NewAlbumGrid(iter, g.im.GetAlbumThumbnail, false)
	g.grid.OnPlayAlbum = g.onPlayAlbum
	g.grid.OnShowArtistPage = g.onShowArtistPage
	g.grid.OnShowAlbumPage = g.onShowAlbumPage
	g.searcher = widgets.NewSearcher()
	g.searcher.OnSearched = g.OnSearched
	searchVbox := container.NewVBox(layout.NewSpacer(), g.searcher.Entry, layout.NewSpacer())
	g.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), g.titleDisp, layout.NewSpacer(), searchVbox, widgets.NewHSpace(15)),
		nil,
		nil,
		nil,
		g.grid,
	)
	return g
}

func (g *GenrePage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

func (a *GenrePage) Route() Route {
	return GenreRoute(a.genre)
}

func (a *GenrePage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (g *GenrePage) Reload() {
	if g.searchText != "" {
		g.doSearch(g.searchText)
	} else {
		g.grid.Reset(g.lm.GenreIter(g.genre))
		g.grid.Refresh()
	}
}

func (a *GenrePage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID, 0)
	}
}

func (a *GenrePage) onShowArtistPage(artistID string) {
	a.nav(ArtistRoute(artistID))
}

func (a *GenrePage) onShowAlbumPage(albumID string) {
	a.nav(AlbumRoute(albumID))
}

func (g *GenrePage) OnSearched(query string) {
	g.searchText = query
	if query == "" {
		g.container.Objects[0] = g.grid
		if g.searchGrid != nil {
			g.searchGrid.Clear()
		}
		g.Refresh()
		return
	}
	g.doSearch(query)
}

func (g *GenrePage) doSearch(query string) {
	iter := g.lm.SearchIterWithFilter(query, func(al *subsonic.AlbumID3) bool {
		return al.Genre == g.genre
	})
	if g.searchGrid == nil {
		g.searchGrid = widgets.NewAlbumGrid(iter, g.im.GetAlbumThumbnail, false /*showYear*/)
		g.searchGrid.OnPlayAlbum = g.onPlayAlbum
		g.searchGrid.OnShowAlbumPage = g.onShowAlbumPage
		g.searchGrid.OnShowArtistPage = g.onShowArtistPage
	} else {
		g.searchGrid.Reset(iter)
	}
	g.container.Objects[0] = g.searchGrid
	g.Refresh()
}
