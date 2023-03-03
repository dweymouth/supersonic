package browsing

import (
	"supersonic/backend"
	"supersonic/sharedutil"
	"supersonic/ui/controller"
	"supersonic/ui/widgets"

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
	contr      controller.Controller
	pm         *backend.PlaybackManager
	im         *backend.ImageManager
	lm         *backend.LibraryManager
	grid       *widgets.AlbumGrid
	searchGrid *widgets.AlbumGrid
	searcher   *widgets.Searcher
	searchText string
	titleDisp  *widget.RichText
	sortOrder  *selectWidget
	container  *fyne.Container
}

type selectWidget struct {
	widget.Select
	height float32
}

func NewSelect(options []string, onChanged func(string)) *selectWidget {
	s := &selectWidget{
		Select: widget.Select{
			Options:   options,
			OnChanged: onChanged,
		},
	}
	s.height = widget.NewSelect(nil, nil).MinSize().Height
	s.ExtendBaseWidget(s)
	return s
}

func (s *selectWidget) MinSize() fyne.Size {
	return fyne.NewSize(170, s.height)
}

func NewAlbumsPage(cfg *backend.AlbumsPageConfig, contr controller.Controller, pm *backend.PlaybackManager, lm *backend.LibraryManager, im *backend.ImageManager) *AlbumsPage {
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
	if !sharedutil.StringSliceContains(backend.AlbumSortOrders, cfg.SortOrder) {
		cfg.SortOrder = string(backend.AlbumSortRecentlyAdded)
	}
	a.sortOrder.Selected = cfg.SortOrder
	iter := lm.AlbumsIter(backend.AlbumSortOrder(a.sortOrder.Selected))
	a.grid = widgets.NewAlbumGrid(iter, im, false /*showYear*/)
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.grid.OnShowArtistPage = a.onShowArtistPage
	a.grid.OnShowAlbumPage = a.onShowAlbumPage
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.OnSearched
	a.createContainer(false)

	return a
}

func (a *AlbumsPage) createContainer(searchgrid bool) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher.Entry, layout.NewSpacer())
	sortVbox := container.NewVBox(layout.NewSpacer(), a.sortOrder, layout.NewSpacer())
	g := a.grid
	if searchgrid {
		g = a.searchGrid
	}
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), a.titleDisp, sortVbox, layout.NewSpacer(), searchVbox, widgets.NewHSpace(15)),
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
	a.grid = widgets.NewAlbumGridFromState(saved.gridState)
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.OnSearched
	a.searcher.Entry.Text = saved.searchText
	if saved.searchText != "" {
		a.searchGrid = widgets.NewAlbumGridFromState(saved.searchGridState)
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
	return a.searcher.Entry
}

func (a *AlbumsPage) Reload() {
	if a.searchText != "" {
		a.doSearch(a.searchText)
	} else {
		a.grid.Reset(a.lm.AlbumsIter(backend.AlbumSortOrder(a.sortOrder.Selected)))
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
		a.searchGrid = widgets.NewAlbumGrid(a.lm.SearchIter(query), a.im, false /*showYear*/)
		a.searchGrid.OnPlayAlbum = a.onPlayAlbum
		a.searchGrid.OnShowAlbumPage = a.onShowAlbumPage
		a.searchGrid.OnShowArtistPage = a.onShowArtistPage
	} else {
		a.searchGrid.Reset(a.lm.SearchIter(query))
	}
	a.container.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *AlbumsPage) onPlayAlbum(albumID string) {
	go a.pm.PlayAlbum(albumID, 0)
}

func (a *AlbumsPage) onShowArtistPage(artistID string) {
	a.contr.NavigateTo(controller.ArtistRoute(artistID))
}

func (a *AlbumsPage) onShowAlbumPage(albumID string) {
	a.contr.NavigateTo(controller.AlbumRoute(albumID))
}

func (a *AlbumsPage) onSortOrderChanged(order string) {
	a.cfg.SortOrder = a.sortOrder.Selected
	a.grid.Reset(a.lm.AlbumsIter(backend.AlbumSortOrder(order)))
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
	contr           controller.Controller
	pm              *backend.PlaybackManager
	lm              *backend.LibraryManager
	im              *backend.ImageManager
	sortOrder       string
	gridState       widgets.AlbumGridState
	searchGridState widgets.AlbumGridState
}

func (s *savedAlbumsPage) Restore() Page {
	return restoreAlbumsPage(s)
}
