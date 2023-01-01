package ui

import (
	"log"
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type ArtistPage struct {
	widget.BaseWidget

	artistID    string
	im          *backend.ImageManager
	sm          *backend.ServerManager
	grid        *AlbumGrid
	titleDisp   *widget.RichText
	container   *fyne.Container
	OnPlayAlbum func(string)
}

func NewArtistPage(artistID string, sm *backend.ServerManager, im *backend.ImageManager) *ArtistPage {
	a := &ArtistPage{
		artistID: artistID,
		sm:       sm,
		im:       im,
	}
	a.ExtendBaseWidget(a)
	a.titleDisp = widget.NewRichTextWithText("Artist")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.container = container.NewBorder(a.titleDisp, nil, nil, nil, layout.NewSpacer())
	a.loadAsync()
	return a
}

func (a *ArtistPage) SetPlayAlbumCallback(cb func(string)) {
	a.OnPlayAlbum = cb
}

func (a *ArtistPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID)
	}
}

func (a *ArtistPage) loadAsync() {
	go func() {
		artist, err := a.sm.Server.GetArtist(a.artistID)
		if err != nil {
			log.Printf("Failed to get artist: %s", err.Error())
			return
		}
		a.titleDisp.Segments[0].(*widget.TextSegment).Text = artist.Name
		a.titleDisp.Refresh()
		ag := NewFixedAlbumGrid(artist.Album, a.im.GetAlbumThumbnail, true /*showYear*/)
		ag.OnPlayAlbum = a.onPlayAlbum
		a.container.Objects[0] = ag
		a.container.Refresh()
	}()
}

func (a *ArtistPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}
