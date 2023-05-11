package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/dialogs"
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

	cfg        *backend.AlbumsPageConfig
	contr      *controller.Controller
	pm         *backend.PlaybackManager
	im         *backend.ImageManager
	lm         *backend.LibraryManager
	grid       *widgets.GridView
	searchGrid *widgets.GridView
	searcher   *widgets.SearchEntry
	filterBtn  *widgets.AlbumFilterButton
	searchText string
	filter     backend.AlbumFilter
	titleDisp  *widget.RichText
	sortOrder  *selectWidget
	container  *fyne.Container
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

func NewAlbumsPage(cfg *backend.AlbumsPageConfig, contr *controller.Controller, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager) *AlbumsPage {
	a := &AlbumsPage{
		cfg:   cfg,
		contr: contr,
		pm:    pm,
		lm:    lm,
		im:    im,
	}
	a.ExtendBaseWidget(a)

	a.titleDisp = widget.NewRichTextWithText("Albums")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.sortOrder = NewSelect(backend.AlbumSortOrders, a.onSortOrderChanged)
	if !sharedutil.SliceContains(backend.AlbumSortOrders, cfg.SortOrder) {
		cfg.SortOrder = string(backend.AlbumSortRecentlyAdded)
	}
	a.sortOrder.Selected = cfg.SortOrder
	iter := lm.AlbumsIter(backend.AlbumSortOrder(a.sortOrder.Selected), a.filter)
	a.grid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(iter), im)
	contr.ConnectAlbumGridActions(a.grid)
	a.searcher = widgets.NewSearchEntry()
	a.searcher.OnSearched = a.OnSearched
	a.filterBtn = widgets.NewAlbumFilterButton(&a.filter, func(filter *backend.AlbumFilter) fyne.CanvasObject { return dialogs.NewAlbumFilterDialog(filter) })
	a.createContainer(false)

	return a
}

func (a *AlbumsPage) createContainer(searchgrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	sortVbox := container.NewVBox(layout.NewSpacer(), a.sortOrder, layout.NewSpacer())
	g := a.grid
	if searchgrid {
		g = a.searchGrid
	}
	a.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6), a.titleDisp, sortVbox, layout.NewSpacer(), a.filterBtn, searchVbox, util.NewHSpace(12)),
		nil,
		nil,
		nil,
		g,
	)
}

func restoreAlbumsPage(saved *savedAlbumsPage) *AlbumsPage {
	a := &AlbumsPage{
		cfg:   saved.cfg,
		contr: saved.contr,
		pm:    saved.pm,
		lm:    saved.lm,
		im:    saved.im,
	}
	a.ExtendBaseWidget(a)

	a.titleDisp = widget.NewRichTextWithText("Albums")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.sortOrder = NewSelect(backend.AlbumSortOrders, nil)
	a.sortOrder.Selected = saved.sortOrder
	a.sortOrder.OnChanged = a.onSortOrderChanged
	a.grid = widgets.NewGridViewFromState(saved.gridState)
	a.searcher = widgets.NewSearchEntry()
	a.searcher.OnSearched = a.OnSearched
	a.searcher.Entry.Text = saved.searchText
	a.searchText = saved.searchText
	if a.searchText != "" {
		a.searchGrid = widgets.NewGridViewFromState(saved.searchGridState)
	}
	a.createContainer(saved.searchText != "")

	return a
}

func (a *AlbumsPage) OnSearched(query string) {
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
		iter := a.lm.AlbumsIter(backend.AlbumSortOrder(a.sortOrder.Selected), a.filter)
		a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
		a.grid.Refresh()
	}
}

func (a *AlbumsPage) Save() SavedPage {
	sa := &savedAlbumsPage{
		cfg:        a.cfg,
		contr:      a.contr,
		pm:         a.pm,
		lm:         a.lm,
		im:         a.im,
		searchText: a.searchText,
		sortOrder:  a.sortOrder.Selected,
		gridState:  a.grid.SaveToState(),
	}
	if a.searchGrid != nil {
		sa.searchGridState = a.searchGrid.SaveToState()
	}
	return sa
}

func (a *AlbumsPage) doSearch(query string) {
	if a.searchGrid == nil {
		a.searchGrid = widgets.NewGridView(widgets.NewGridViewAlbumIterator(a.lm.SearchIter(query)), a.im)
		a.contr.ConnectAlbumGridActions(a.searchGrid)
	} else {
		a.searchGrid.Reset(widgets.NewGridViewAlbumIterator(a.lm.SearchIter(query)))
	}
	a.container.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *AlbumsPage) onSortOrderChanged(order string) {
	a.cfg.SortOrder = a.sortOrder.Selected
	iter := a.lm.AlbumsIter(backend.AlbumSortOrder(order), a.filter)
	a.grid.Reset(widgets.NewGridViewAlbumIterator(iter))
	if a.searchText == "" {
		a.container.Objects[0] = a.grid
		a.Refresh()
	}
}

func (a *AlbumsPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedAlbumsPage struct {
	searchText      string
	cfg             *backend.AlbumsPageConfig
	contr           *controller.Controller
	pm              *backend.PlaybackManager
	lm              *backend.LibraryManager
	im              *backend.ImageManager
	sortOrder       string
	gridState       widgets.GridViewState
	searchGridState widgets.GridViewState
}

func (s *savedAlbumsPage) Restore() Page {
	return restoreAlbumsPage(s)
}
