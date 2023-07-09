package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
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

	genre           string
	pool            *util.WidgetPool
	contr           *controller.Controller
	im              *backend.ImageManager
	pm              *backend.PlaybackManager
	mp              mediaprovider.MediaProvider
	grid            *widgets.GridView
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
	searcher        *widgets.SearchEntry
	searchText      string
	filter          mediaprovider.AlbumFilter
	filterBtn       *widgets.AlbumFilterButton
	titleDisp       *widget.RichText
	playRandom      *widget.Button

	container *fyne.Container
}

func NewGenrePage(genre string, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) *GenrePage {
	g := &GenrePage{
		genre:  genre,
		pool:   pool,
		filter: mediaprovider.AlbumFilter{Genres: []string{genre}},
		contr:  contr,
		pm:     pm,
		mp:     mp,
		im:     im,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	iter := widgets.NewGridViewAlbumIterator(g.mp.IterateAlbums("", g.filter))
	if gv := pool.Obtain(util.WidgetTypeGridView); gv != nil {
		g.grid = gv.(*widgets.GridView)
		g.grid.Placeholder = myTheme.AlbumIcon
		g.grid.Reset(iter)
	} else {
		g.grid = widgets.NewGridView(iter, g.im, myTheme.AlbumIcon)
	}
	g.contr.ConnectAlbumGridActions(g.grid)
	g.createSearchAndFilter()
	g.createContainer()

	return g
}

func (g *GenrePage) createSearchAndFilter() {
	g.searcher = widgets.NewSearchEntry()
	g.searcher.Text = g.searchText
	g.searcher.OnSearched = g.OnSearched
	// genre filter is disabled for this page, so no need to actually call genre list fetching function
	g.filterBtn = widgets.NewAlbumFilterButton(&g.filter, func() ([]*mediaprovider.Genre, error) { return nil, nil })
	g.filterBtn.GenreDisabled = true
	g.filterBtn.OnChanged = g.Reload
}

func (g *GenrePage) createContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), g.searcher, layout.NewSpacer())
	playRandomVbox := container.NewVBox(layout.NewSpacer(), g.playRandom, layout.NewSpacer())
	g.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6),
			g.titleDisp, playRandomVbox, layout.NewSpacer(), container.NewCenter(g.filterBtn), searchVbox, util.NewHSpace(15)),
		nil, nil, nil, g.grid,
	)
}

func restoreGenrePage(saved *savedGenrePage) *GenrePage {
	g := &GenrePage{
		genre:           saved.genre,
		pool:            saved.pool,
		contr:           saved.contr,
		pm:              saved.pm,
		mp:              saved.mp,
		im:              saved.im,
		gridState:       saved.gridState,
		searchGridState: saved.searchGridState,
		searchText:      saved.searchText,
		filter:          saved.filter,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(g.genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	state := saved.gridState
	if g.searchText != "" {
		state = saved.searchGridState
	}
	if gv := g.pool.Obtain(util.WidgetTypeGridView); gv != nil {
		g.grid = gv.(*widgets.GridView)
		g.grid.Placeholder = myTheme.AlbumIcon
		g.grid.ResetFromState(state)
	} else {
		g.grid = widgets.NewGridViewFromState(state)
	}
	g.createSearchAndFilter()
	g.createContainer()

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
		iter := g.mp.IterateAlbums("", g.filter)
		g.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
	}
}

func (g *GenrePage) Save() SavedPage {
	sg := &savedGenrePage{
		genre:           g.genre,
		pool:            g.pool,
		filter:          g.filter,
		searchText:      g.searchText,
		contr:           g.contr,
		pm:              g.pm,
		mp:              g.mp,
		im:              g.im,
		gridState:       g.gridState,
		searchGridState: g.searchGridState,
	}
	if g.searchText != "" {
		sg.searchGridState = g.grid.SaveToState()
	} else {
		sg.gridState = g.grid.SaveToState()
	}
	g.grid.Clear()
	g.pool.Release(util.WidgetTypeGridView, g.grid)
	return sg
}

var _ Searchable = (*AlbumsPage)(nil)

func (g *GenrePage) SearchWidget() fyne.Focusable {
	return g.searcher
}

func (g *GenrePage) OnSearched(query string) {
	if query == "" {
		g.grid.ResetFromState(g.gridState)
	} else {
		g.doSearch(query)
	}
	g.searchText = query
}

func (g *GenrePage) doSearch(query string) {
	if g.searchText == "" {
		g.gridState = g.grid.SaveToState()
	}
	iter := g.mp.SearchAlbums(query, g.filter)
	g.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
}

func (g *GenrePage) playRandomSongs() {
	go g.pm.PlayRandomSongs(g.genre)
}

type savedGenrePage struct {
	genre           string
	searchText      string
	pool            *util.WidgetPool
	filter          mediaprovider.AlbumFilter
	contr           *controller.Controller
	pm              *backend.PlaybackManager
	mp              mediaprovider.MediaProvider
	im              *backend.ImageManager
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
}

func (s *savedGenrePage) Restore() Page {
	return restoreGenrePage(s)
}
