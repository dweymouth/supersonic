package ui

import (
	"supersonic/backend"
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

	im          *backend.ImageManager
	lm          *backend.LibraryManager
	grid        *AlbumGrid
	searchGrid  *AlbumGrid
	searchText  string
	titleDisp   *widget.RichText
	sortOrder   *selectWidget
	container   *fyne.Container
	OnPlayAlbum func(string)
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

func NewAlbumsPage(title string, lm *backend.LibraryManager, im *backend.ImageManager) *AlbumsPage {
	a := &AlbumsPage{
		lm: lm,
		im: im,
	}
	a.ExtendBaseWidget(a)

	a.titleDisp = widget.NewRichTextWithText(title)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.sortOrder = NewSelect(backend.AlbumSortOrders, nil)
	a.sortOrder.Selected = a.sortOrder.Options[0]
	a.sortOrder.OnChanged = a.onSortOrderChanged
	sortVbox := container.NewVBox(layout.NewSpacer(), a.sortOrder, layout.NewSpacer())
	a.grid = NewAlbumGrid(lm.AlbumsIter(backend.AlbumSortOrder(a.sortOrder.Selected)), im.GetAlbumThumbnail)
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(15), a.titleDisp, sortVbox),
		nil,
		nil,
		nil,
		a.grid,
	)
	return a
}

func (a *AlbumsPage) OnSearched(query string) {
	a.searchText = query
	if query == "" {
		a.container.Objects[0] = a.grid
		a.searchGrid = nil
		a.Refresh()
		return
	}
	a.searchGrid = NewAlbumGrid(a.lm.SearchIter(query), a.im.GetAlbumThumbnail)
	a.searchGrid.OnPlayAlbum = a.onPlayAlbum
	a.container.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *AlbumsPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID)
	}
}

func (a *AlbumsPage) onSortOrderChanged(order string) {
	a.grid = NewAlbumGrid(a.lm.AlbumsIter(backend.AlbumSortOrder(order)), a.im.GetAlbumThumbnail)
	if a.searchText == "" {
		a.container.Objects[0] = a.grid
		a.Refresh()
	}
}

func (a *AlbumsPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}
