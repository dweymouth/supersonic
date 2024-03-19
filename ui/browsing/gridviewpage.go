package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Base widget for grid view pages
type GridViewPage[M, F any] struct {
	widget.BaseWidget

	adapter GridViewPageAdapter[M, F]
	pool    *util.WidgetPool
	mp      mediaprovider.MediaProvider
	im      *backend.ImageManager

	grid            *widgets.GridView
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState

	title      *widget.RichText
	sortOrder  *sortOrderSelect
	filterBtn  widgets.FilterButton[M, F]
	filter     mediaprovider.MediaFilter[M, F]
	searcher   *widgets.SearchEntry
	searchText string

	container *fyne.Container
}

// Base type for pages that show an iterable GridView
type GridViewPageAdapter[M, F any] interface {
	// Returns the title for the page
	Title() string

	// Returns the base media filter for this page, if any.
	// A filterable page with no base filters applied should return a zero-valued
	// filter pointer, *not* nil. (Nil means unfilterable and no filter button created.)
	Filter() mediaprovider.MediaFilter[M, F]

	// Returns the filter button for the page, if any.
	FilterButton() widgets.FilterButton[M, F]

	// Returns the cover placeholder resource for the page
	PlaceholderResource() fyne.Resource

	// Returns the route for the page
	Route() controller.Route

	// Returns the ActionButton for this page, if any
	ActionButton() *widget.Button

	// Returns the iterator for the given sortOrder and filter.
	// (Non-media pages can ignore the filter argument)
	Iter(sortOrder string, filter mediaprovider.MediaFilter[M, F]) widgets.GridViewIterator

	// Returns the iterator for the given search query and filter.
	SearchIter(query string, filter mediaprovider.MediaFilter[M, F]) widgets.GridViewIterator

	// Function that connects the GridView callbacks to the appropriate action handlers.
	ConnectGridActions(*widgets.GridView)
}

type SortableGridViewPageAdapter interface {
	// Returns the list of sort orders and the initially selected sort order
	SortOrders() ([]string, string)

	// Saves the given sort order setting.
	SaveSortOrder(string)
}

type sortOrderSelect struct {
	widget.Select
}

func NewSortOrderSelect(options []string, onChanged func(string)) *sortOrderSelect {
	s := &sortOrderSelect{
		Select: widget.Select{
			Options:   options,
			OnChanged: onChanged,
		},
	}
	s.ExtendBaseWidget(s)
	return s
}

func (s *sortOrderSelect) MinSize() fyne.Size {
	return fyne.NewSize(170, s.Select.MinSize().Height)
}

func NewGridViewPage[M, F any](
	adapter GridViewPageAdapter[M, F],
	pool *util.WidgetPool,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
) *GridViewPage[M, F] {
	gp := &GridViewPage[M, F]{
		adapter:   adapter,
		pool:      pool,
		mp:        mp,
		im:        im,
		filter:    adapter.Filter(),
		filterBtn: adapter.FilterButton(),
	}
	gp.ExtendBaseWidget(gp)
	gp.createTitleAndSort()

	_, canShare := mp.(mediaprovider.SupportsSharing)
	iter := adapter.Iter(gp.getSortOrder(), gp.getFilter())
	if g := pool.Obtain(util.WidgetTypeGridView); g != nil {
		gp.grid = g.(*widgets.GridView)
		gp.grid.Placeholder = adapter.PlaceholderResource()
		gp.grid.Reset(iter)
	} else {
		gp.grid = widgets.NewGridView(iter, im, adapter.PlaceholderResource())
	}
	gp.grid.DisableSharing = !canShare
	adapter.ConnectGridActions(gp.grid)
	gp.createSearchAndFilter()
	gp.createContainer()
	return gp
}

func (g *GridViewPage[M, F]) createTitleAndSort() {
	g.title = widget.NewRichText(&widget.TextSegment{
		Text:  g.adapter.Title(),
		Style: widget.RichTextStyle{SizeName: theme.SizeNameHeadingText},
	})
	if s, ok := g.adapter.(SortableGridViewPageAdapter); ok {
		sorts, selected := s.SortOrders()
		g.sortOrder = NewSortOrderSelect(sorts, g.onSortOrderChanged)
		g.sortOrder.Selected = selected
	}
}

func (g *GridViewPage[M, F]) createSearchAndFilter() {
	g.searcher = widgets.NewSearchEntry()
	g.searcher.PlaceHolder = "Search page"
	g.searcher.Text = g.searchText
	g.searcher.OnSearched = g.OnSearched
	if g.filterBtn != nil {
		g.filterBtn.SetOnChanged(g.Reload)
	}
}

func (g *GridViewPage[M, F]) createContainer() {
	header := container.NewHBox(util.NewHSpace(6), g.title)
	if g.sortOrder != nil {
		header.Add(container.NewCenter(g.sortOrder))
	}
	if b := g.adapter.ActionButton(); b != nil {
		header.Add(container.NewCenter(b))
	}
	header.Add(layout.NewSpacer())
	if g.filterBtn != nil {
		header.Add(container.NewCenter(g.filterBtn))
	}
	header.Add(container.NewCenter(g.searcher))
	header.Add(util.NewHSpace(12))
	g.container = container.NewBorder(header, nil, nil, nil, g.grid)
}

func (g *GridViewPage[M, F]) Reload() {
	if g.searchText != "" {
		g.doSearch(g.searchText)
	} else {
		g.grid.Reset(g.adapter.Iter(g.getSortOrder(), g.getFilter()))
	}
}

func (g *GridViewPage[M, F]) Route() controller.Route {
	return g.adapter.Route()
}

func (g *GridViewPage[M, F]) SearchWidget() fyne.Focusable {
	return g.searcher
}

func (g *GridViewPage[M, F]) Scroll(scrollAmt float32) {
	g.grid.ScrollToOffset(g.grid.GetScrollOffset() + scrollAmt)
}

func (g *GridViewPage[M, F]) OnSearched(query string) {
	if query == "" {
		if g.sortOrder != nil {
			g.sortOrder.Enable()
		}
		g.grid.ResetFromState(g.gridState)
		g.searchGridState = nil
	} else {
		if g.sortOrder != nil {
			g.sortOrder.Disable()
		}
		g.doSearch(query)
	}
	g.searchText = query
}

func (g *GridViewPage[M, F]) doSearch(query string) {
	if g.searchText == "" {
		g.gridState = g.grid.SaveToState()
	}
	g.grid.Reset(g.adapter.SearchIter(query, g.getFilter()))
}

func (g *GridViewPage[M, F]) onSortOrderChanged(order string) {
	g.adapter.(SortableGridViewPageAdapter).SaveSortOrder(g.getSortOrder())
	g.grid.Reset(g.adapter.Iter(g.getSortOrder(), g.getFilter()))
}

func (g *GridViewPage[M, F]) getFilter() mediaprovider.MediaFilter[M, F] {
	return g.filter
}

func (g *GridViewPage[M, F]) getSortOrder() string {
	if g.sortOrder != nil {
		return g.sortOrder.Selected
	}
	return ""
}

func (g *GridViewPage[M, F]) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

type savedGridViewPage[M, F any] struct {
	adapter         GridViewPageAdapter[M, F]
	im              *backend.ImageManager
	mp              mediaprovider.MediaProvider
	searchText      string
	filter          mediaprovider.MediaFilter[M, F]
	pool            *util.WidgetPool
	sortOrder       string
	gridState       *widgets.GridViewState
	searchGridState *widgets.GridViewState
}

func (g *GridViewPage[M, F]) Save() SavedPage {
	sa := &savedGridViewPage[M, F]{
		adapter:         g.adapter,
		pool:            g.pool,
		mp:              g.mp,
		im:              g.im,
		searchText:      g.searchText,
		filter:          g.filter,
		sortOrder:       g.getSortOrder(),
		gridState:       g.gridState,
		searchGridState: g.searchGridState,
	}
	if g.searchText == "" {
		sa.gridState = g.grid.SaveToState()
	} else {
		sa.searchGridState = g.grid.SaveToState()
	}
	g.grid.Clear()
	g.pool.Release(util.WidgetTypeGridView, g.grid)
	return sa
}

func (s *savedGridViewPage[M, F]) Restore() Page {
	gp := &GridViewPage[M, F]{
		adapter:         s.adapter,
		pool:            s.pool,
		mp:              s.mp,
		im:              s.im,
		gridState:       s.gridState,
		searchGridState: s.searchGridState,
		searchText:      s.searchText,
		filter:          s.filter,
	}
	gp.ExtendBaseWidget(gp)

	gp.createTitleAndSort()
	state := s.gridState
	if s.searchText != "" {
		if gp.sortOrder != nil {
			gp.sortOrder.Disable()
		}
		state = s.searchGridState
	}
	if g := gp.pool.Obtain(util.WidgetTypeGridView); g != nil {
		gp.grid = g.(*widgets.GridView)
		gp.grid.ResetFromState(state)
	} else {
		gp.grid = widgets.NewGridViewFromState(state)
	}
	gp.adapter.ConnectGridActions(gp.grid)
	gp.createSearchAndFilter()
	gp.createContainer()
	return gp
}
