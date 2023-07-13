package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
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

var _ fyne.Widget = (*AlbumsPage)(nil)

type AlbumsPage struct {
	widget.BaseWidget

	cfg             *backend.AlbumsPageConfig
	pool            *util.WidgetPool
	contr           *controller.Controller
	pm              *backend.PlaybackManager
	im              *backend.ImageManager
	mp              mediaprovider.MediaProvider
	grid            *widgets.GridView
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
	searcher        *widgets.SearchEntry
	filterBtn       *widgets.AlbumFilterButton
	searchText      string
	filter          mediaprovider.AlbumFilter
	titleDisp       *widget.RichText
	sortOrder       *selectWidget
	container       *fyne.Container
}

type selectWidget struct {
	widget.Select
}

func NewSelect(options []string, onChanged func(string)) *selectWidget {
	s := &selectWidget{
		Select: widget.Select{
			Options:   options,
			OnChanged: onChanged,
		},
	}
	s.ExtendBaseWidget(s)
	return s
}

func (s *selectWidget) MinSize() fyne.Size {
	return fyne.NewSize(170, s.Select.MinSize().Height)
}

func NewAlbumsPage(cfg *backend.AlbumsPageConfig, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) *AlbumsPage {
	a := &AlbumsPage{
		pool:  pool,
		cfg:   cfg,
		contr: contr,
		pm:    pm,
		mp:    mp,
		im:    im,
	}
	a.ExtendBaseWidget(a)

	a.titleDisp = widget.NewRichTextWithText("Albums")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.sortOrder = NewSelect(mp.AlbumSortOrders(), a.onSortOrderChanged)
	if !sharedutil.SliceContains(mp.AlbumSortOrders(), cfg.SortOrder) {
		cfg.SortOrder = string(mp.AlbumSortOrders()[0])
	}
	a.sortOrder.Selected = cfg.SortOrder
	iter := mp.IterateAlbums(a.sortOrder.Selected, a.filter)
	if g := pool.Obtain(util.WidgetTypeGridView); g != nil {
		a.grid = g.(*widgets.GridView)
		a.grid.Placeholder = myTheme.AlbumIcon
		a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
	} else {
		a.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), im, myTheme.AlbumIcon)
	}
	contr.ConnectAlbumGridActions(a.grid)
	a.createSearchAndFilter()
	a.createContainer()

	return a
}

func (a *AlbumsPage) createSearchAndFilter() {
	a.searcher = widgets.NewSearchEntry()
	a.searcher.Text = a.searchText
	a.searcher.OnSearched = a.OnSearched
	a.filterBtn = widgets.NewAlbumFilterButton(&a.filter, a.mp.GetGenres)
	a.filterBtn.OnChanged = a.Reload
}

func (a *AlbumsPage) createContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	sortVbox := container.NewVBox(layout.NewSpacer(), a.sortOrder, layout.NewSpacer())
	a.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6), a.titleDisp, sortVbox, layout.NewSpacer(), container.NewCenter(a.filterBtn), searchVbox, util.NewHSpace(12)),
		nil,
		nil,
		nil,
		a.grid,
	)
}

func restoreAlbumsPage(saved *savedAlbumsPage) *AlbumsPage {
	a := &AlbumsPage{
		cfg:             saved.cfg,
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
	a.ExtendBaseWidget(a)

	a.titleDisp = widget.NewRichTextWithText("Albums")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.sortOrder = NewSelect(a.mp.AlbumSortOrders(), nil)
	a.sortOrder.Selected = saved.sortOrder
	a.sortOrder.OnChanged = a.onSortOrderChanged
	state := saved.gridState
	if a.searchText != "" {
		a.sortOrder.Disable()
		state = saved.searchGridState
	}
	if g := a.pool.Obtain(util.WidgetTypeGridView); g != nil {
		a.grid = g.(*widgets.GridView)
		a.grid.ResetFromState(state)
	} else {
		a.grid = widgets.NewGridViewFromState(state)
	}
	a.createSearchAndFilter()
	a.createContainer()

	return a
}

func (a *AlbumsPage) OnSearched(query string) {
	if query == "" {
		a.sortOrder.Enable()
		a.grid.ResetFromState(a.gridState)
		a.searchGridState = nil
	} else {
		a.sortOrder.Disable()
		a.doSearch(query)
	}
	a.searchText = query
}

func (a *AlbumsPage) Route() controller.Route {
	return controller.AlbumsRoute()
}

var _ Searchable = (*AlbumsPage)(nil)

func (a *AlbumsPage) SearchWidget() fyne.Focusable {
	return a.searcher
}

func (a *AlbumsPage) Reload() {
	if a.searchText != "" {
		a.doSearch(a.searchText)
	} else {
		iter := a.mp.IterateAlbums(a.sortOrder.Selected, a.filter)
		a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
	}
}

func (a *AlbumsPage) Save() SavedPage {
	sa := &savedAlbumsPage{
		cfg:             a.cfg,
		pool:            a.pool,
		contr:           a.contr,
		pm:              a.pm,
		mp:              a.mp,
		im:              a.im,
		searchText:      a.searchText,
		filter:          a.filter,
		sortOrder:       a.sortOrder.Selected,
		gridState:       a.gridState,
		searchGridState: a.searchGridState,
	}
	if a.searchText == "" {
		sa.gridState = a.grid.SaveToState()
	} else {
		sa.searchGridState = a.grid.SaveToState()
	}
	a.grid.Clear()
	a.pool.Release(util.WidgetTypeGridView, a.grid)
	return sa
}

func (a *AlbumsPage) doSearch(query string) {
	if a.searchText == "" {
		a.gridState = a.grid.SaveToState()
	}
	iter := widgets.NewGridViewAlbumIterator(a.mp.SearchAlbums(query, a.filter))
	a.grid.Reset(iter)
}

func (a *AlbumsPage) onSortOrderChanged(order string) {
	a.cfg.SortOrder = a.sortOrder.Selected
	iter := a.mp.IterateAlbums(order, a.filter)
	a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
}

func (a *AlbumsPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedAlbumsPage struct {
	searchText      string
	filter          mediaprovider.AlbumFilter
	cfg             *backend.AlbumsPageConfig
	pool            *util.WidgetPool
	contr           *controller.Controller
	pm              *backend.PlaybackManager
	mp              mediaprovider.MediaProvider
	im              *backend.ImageManager
	sortOrder       string
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
}

func (s *savedAlbumsPage) Restore() Page {
	return restoreAlbumsPage(s)
}
