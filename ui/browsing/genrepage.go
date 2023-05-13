package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TODO: there is a lot of code duplication between this and albumspage. Refactor?
type GenrePage struct {
	widget.BaseWidget

	genre      string
	contr      *controller.Controller
	im         *backend.ImageManager
	pm         *backend.PlaybackManager
	lm         *backend.LibraryManager
	grid       *widgets.GridView
	searchGrid *widgets.GridView
	searcher   *widgets.SearchEntry
	searchText string
	filter     backend.AlbumFilter
	filterBtn  *widgets.AlbumFilterButton
	titleDisp  *widget.RichText
	playRandom *widget.Button

	OnPlayAlbum func(string, int)

	container *fyne.Container
}

func NewGenrePage(genre string, contr *controller.Controller, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager) *GenrePage {
	g := &GenrePage{
		genre:  genre,
		filter: backend.AlbumFilter{Genres: []string{genre}},
		contr:  contr,
		pm:     pm,
		lm:     lm,
		im:     im,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	iter := g.lm.GenreIter(g.genre, g.filter)
	g.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), g.im)
	g.contr.ConnectAlbumGridActions(g.grid)
	g.createSearchAndFilter()
	g.createContainer(false)

	return g
}

func (g *GenrePage) createSearchAndFilter() {
	g.searcher = widgets.NewSearchEntry()
	g.searcher.Text = g.searchText
	g.searcher.OnSearched = g.OnSearched
	g.filterBtn = widgets.NewAlbumFilterButton(&g.filter)
	g.filterBtn.GenreDisabled = true
	g.filterBtn.OnChanged = g.Reload
}

func (g *GenrePage) createContainer(searchGrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), g.searcher, layout.NewSpacer())
	gr := g.grid
	if searchGrid {
		gr = g.searchGrid
	}
	playRandomVbox := container.NewVBox(layout.NewSpacer(), g.playRandom, layout.NewSpacer())
	g.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6),
			g.titleDisp, playRandomVbox, layout.NewSpacer(), container.NewCenter(g.filterBtn), searchVbox, util.NewHSpace(15)),
		nil, nil, nil, gr,
	)
}

func restoreGenrePage(saved *savedGenrePage) *GenrePage {
	g := &GenrePage{
		genre:      saved.genre,
		contr:      saved.contr,
		pm:         saved.pm,
		lm:         saved.lm,
		im:         saved.im,
		searchText: saved.searchText,
		filter:     saved.filter,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(g.genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	g.grid = widgets.NewGridViewFromState(saved.gridState)
	g.createSearchAndFilter()
	if g.searchText != "" {
		g.searchGrid = widgets.NewGridViewFromState(saved.searchGridState)
	}
	g.createContainer(saved.searchText != "")

	return g
}

func (g *GenrePage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

func (a *GenrePage) Route() controller.Route {
	return controller.GenreRoute(a.genre)
}

func (g *GenrePage) Reload() {
	if g.searchText != "" {
		g.doSearch(g.searchText)
	} else {
		iter := g.lm.GenreIter(g.genre, g.filter)
		g.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
		g.grid.Refresh()
	}
}

func (g *GenrePage) Save() SavedPage {
	sg := &savedGenrePage{
		genre:      g.genre,
		filter:     g.filter,
		searchText: g.searchText,
		contr:      g.contr,
		pm:         g.pm,
		lm:         g.lm,
		im:         g.im,
		gridState:  g.grid.SaveToState(),
	}
	if g.searchGrid != nil {
		sg.searchGridState = g.searchGrid.SaveToState()
	}
	return sg
}

var _ Searchable = (*AlbumsPage)(nil)

func (g *GenrePage) SearchWidget() fyne.Focusable {
	return g.searcher
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
	iter := g.lm.SearchIterWithFilter(query, g.filter)
	if g.searchGrid == nil {
		g.searchGrid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), g.im)
		g.contr.ConnectAlbumGridActions(g.searchGrid)
	} else {
		g.searchGrid.Reset(widgets.NewGridViewAlbumIterator(iter))
	}
	g.container.Objects[0] = g.searchGrid
	g.Refresh()
}

func (g *GenrePage) playRandomSongs() {
	go g.pm.PlayRandomSongs(g.genre)
}

type savedGenrePage struct {
	genre           string
	searchText      string
	filter          backend.AlbumFilter
	contr           *controller.Controller
	pm              *backend.PlaybackManager
	lm              *backend.LibraryManager
	im              *backend.ImageManager
	gridState       widgets.GridViewState
	searchGridState widgets.GridViewState
}

func (s *savedGenrePage) Restore() Page {
	return restoreGenrePage(s)
}
