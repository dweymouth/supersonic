package browsing

import (
	"supersonic/backend"
	"supersonic/ui/controller"
	myTheme "supersonic/ui/theme"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
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
	searcher   *widgets.Searcher
	searchText string
	titleDisp  *widget.RichText
	playRandom *widget.Button

	OnPlayAlbum func(string, int)

	container *fyne.Container
}

func NewGenrePage(genre string, contr *controller.Controller, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager) *GenrePage {
	g := &GenrePage{
		genre: genre,
		contr: contr,
		pm:    pm,
		lm:    lm,
		im:    im,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	iter := g.lm.GenreIter(g.genre)
	g.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), g.im)
	g.contr.ConnectAlbumGridActions(g.grid)
	g.searcher = widgets.NewSearcher()
	g.searcher.OnSearched = g.OnSearched
	g.createContainer(false)

	return g
}

func (g *GenrePage) createContainer(searchGrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), g.searcher.Entry, layout.NewSpacer())
	gr := g.grid
	if searchGrid {
		gr = g.searchGrid
	}
	playRandomVbox := container.NewVBox(layout.NewSpacer(), g.playRandom, layout.NewSpacer())
	g.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6), g.titleDisp, playRandomVbox, layout.NewSpacer(), searchVbox, util.NewHSpace(15)),
		nil,
		nil,
		nil,
		gr,
	)
}

func restoreGenrePage(saved *savedGenrePage) *GenrePage {
	g := &GenrePage{
		genre: saved.genre,
		contr: saved.contr,
		pm:    saved.pm,
		lm:    saved.lm,
		im:    saved.im,
	}
	g.ExtendBaseWidget(g)

	g.titleDisp = widget.NewRichTextWithText(g.genre)
	g.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	g.playRandom = widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, g.playRandomSongs)
	g.grid = widgets.NewGridViewFromState(saved.gridState)
	g.searcher = widgets.NewSearcher()
	g.searcher.OnSearched = g.OnSearched
	g.searcher.Entry.Text = saved.searchText
	g.searchText = saved.searchText
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
		g.grid.Reset(widgets.NewGridViewAlbumIterator(g.lm.GenreIter(g.genre)))
		g.grid.Refresh()
	}
}

func (g *GenrePage) Save() SavedPage {
	sg := &savedGenrePage{
		genre:      g.genre,
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
	return g.searcher.Entry
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
