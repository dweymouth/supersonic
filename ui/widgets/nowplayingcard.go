package widgets

import (
	"image"

	"supersonic/ui/layout"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type NowPlayingCard struct {
	widget.BaseWidget

	trackName  *widget.Label
	artistName *CustomHyperlink
	albumName  *CustomHyperlink
	cover      *canvas.Image

	c fyne.CanvasObject
}

func NewNowPlayingCard() *NowPlayingCard {
	n := &NowPlayingCard{
		trackName:  widget.NewLabel(""),
		artistName: NewCustomHyperlink(),
		albumName:  NewCustomHyperlink(),
		cover:      &canvas.Image{},
	}
	n.ExtendBaseWidget(n)
	n.trackName.Wrapping = fyne.TextTruncate
	n.trackName.TextStyle = fyne.TextStyle{Bold: true}
	n.cover.SetMinSize(fyne.NewSize(100, 100))
	n.cover.FillMode = canvas.ImageFillContain

	n.c = container.NewBorder(nil, nil, n.cover, nil, container.New(&layout.VboxCustomPadding{ExtraPad: -10}, n.trackName, n.artistName, n.albumName))
	return n
}

func (n *NowPlayingCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.c)
}

func (n *NowPlayingCard) Update(track, artist, album string, cover image.Image) {
	n.trackName.Text = track
	n.artistName.SetText(artist)
	n.albumName.SetText(album)
	n.cover.Image = cover
	n.c.Refresh()
}

func (n *NowPlayingCard) OnArtistNameTapped(f func()) {
	n.artistName.OnTapped = f
}

func (n *NowPlayingCard) OnAlbumNameTapped(f func()) {
	n.albumName.OnTapped = f
}
