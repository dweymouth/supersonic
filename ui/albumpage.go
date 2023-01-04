package ui

import (
	"log"
	"supersonic/backend"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AlbumPage struct {
	widget.BaseWidget

	albumID   string
	im        *backend.ImageManager
	lm        *backend.LibraryManager
	titleDisp *widget.RichText
	tracklist *widgets.Tracklist
	container *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewAlbumPage(albumID string, lm *backend.LibraryManager) *AlbumPage {
	a := &AlbumPage{albumID: albumID, lm: lm}
	a.ExtendBaseWidget(a)
	a.titleDisp = widget.NewRichTextWithText("Artist")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.container = container.NewBorder(a.titleDisp, nil, nil, nil, layout.NewSpacer())
	a.loadAsync()
	return a
}

func (a *AlbumPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *AlbumPage) Route() Route {
	return AlbumRoute(a.albumID)
}

func (a *AlbumPage) onPlayTrackAt(tracknum int) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(a.albumID, tracknum)
	}
}

func (a *AlbumPage) loadAsync() {
	go func() {
		album, err := a.lm.GetAlbum(a.albumID)
		if err != nil {
			log.Printf("Failed to get album: %s", err.Error())
			return
		}
		a.titleDisp.Segments[0].(*widget.TextSegment).Text = album.Name
		a.titleDisp.Refresh()
		tl := widgets.NewTracklist(album.Song)
		tl.OnPlayTrackAt = a.onPlayTrackAt
		a.container.Objects[0] = tl
		a.container.Refresh()
	}()
}
