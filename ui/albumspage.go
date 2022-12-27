package ui

import (
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*AlbumsPage)(nil)

type AlbumsPage struct {
	widget.BaseWidget

	im            *backend.ImageManager
	lm            *backend.LibraryManager
	grid          *AlbumGrid
	searchGrid    *AlbumGrid
	gridContainer *fyne.Container
	OnPlayAlbum   func(string)
}

func NewAlbumsPage(title string, lm *backend.LibraryManager, im *backend.ImageManager) *AlbumsPage {
	a := &AlbumsPage{
		lm: lm,
		im: im,
	}
	a.ExtendBaseWidget(a)
	a.grid = NewAlbumGrid(lm.RecentlyAddedIter(), im.GetAlbumThumbnail)
	a.grid.OnPlayAlbum = a.onPlayAlbum
	a.gridContainer = container.NewMax(a.grid)
	return a
}

func (a *AlbumsPage) OnSearched(query string) {
	if query == "" {
		a.gridContainer.Objects[0] = a.grid
		a.searchGrid = nil
		a.Refresh()
		return
	}
	a.searchGrid = NewAlbumGrid(a.lm.SearchIter(query), a.im.GetAlbumThumbnail)
	a.searchGrid.OnPlayAlbum = a.onPlayAlbum
	a.gridContainer.Objects[0] = a.searchGrid
	a.Refresh()
}

func (a *AlbumsPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID)
	}
}

func (a *AlbumsPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.gridContainer)
}
