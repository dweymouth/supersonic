package ui

import (
	"fmt"
	"log"
	"supersonic/backend"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type AlbumPage struct {
	widget.BaseWidget

	albumID   string
	im        *backend.ImageManager
	lm        *backend.LibraryManager
	nav       func(Route)
	header    *AlbumPageHeader
	tracklist *widgets.Tracklist
	container *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewAlbumPage(albumID string, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *AlbumPage {
	a := &AlbumPage{albumID: albumID, lm: lm, im: im, nav: nav}
	a.ExtendBaseWidget(a)
	a.header = NewAlbumPageHeader(nav)
	a.container = container.NewBorder(a.header, nil, nil, nil, layout.NewSpacer())
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
		a.header.Update(album, a.im)
		tl := widgets.NewTracklist(album.Song)
		tl.OnPlayTrackAt = a.onPlayTrackAt
		a.container.Objects[0] = tl
		a.container.Refresh()
	}()
}

type AlbumPageHeader struct {
	widget.BaseWidget

	albumID  string
	artistID string

	cover       *canvas.Image
	titleLabel  *widget.RichText
	artistLabel *widgets.CustomHyperlink
	genreLabel  *widget.Label // later custom hyperlink
	miscLabel   *widget.Label

	container *fyne.Container
}

func NewAlbumPageHeader(nav func(Route)) *AlbumPageHeader {
	a := &AlbumPageHeader{}
	a.ExtendBaseWidget(a)
	a.cover = &canvas.Image{FillMode: canvas.ImageFillContain}
	a.cover.SetMinSize(fyne.NewSize(225, 225))
	a.titleLabel = widget.NewRichTextWithText("Album Title")
	a.titleLabel.Wrapping = fyne.TextTruncate
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistLabel = widgets.NewCustomHyperlink()
	a.artistLabel.OnTapped = func() {
		nav(ArtistRoute(a.artistID))
	}
	a.genreLabel = widget.NewLabel("")
	a.miscLabel = widget.NewLabel("")

	a.container = container.NewBorder(nil, nil, a.cover, nil,
		container.NewVBox(a.titleLabel, a.artistLabel, a.genreLabel, a.miscLabel),
	)
	return a
}

func (a *AlbumPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPageHeader) Update(album *subsonic.AlbumID3, im *backend.ImageManager) {
	a.albumID = album.ID
	a.artistID = album.ArtistID
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = album.Name
	a.artistLabel.SetText(album.Artist)
	a.genreLabel.SetText(album.Genre)
	a.miscLabel.SetText(formatMiscLabelStr(album))
	a.Refresh()
	go func() {
		if cover, err := im.GetAlbumThumbnail(album.ID); err == nil {
			a.cover.Image = cover
			a.cover.Refresh()
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
}

func formatMiscLabelStr(a *subsonic.AlbumID3) string {
	return fmt.Sprintf("%d · %d tracks · %s", a.Year, a.SongCount, util.SecondsToTimeString(float64(a.Duration)))
}
