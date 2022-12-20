package ui

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
	artistName *widget.Label
	albumName  *widget.Label
	cover      *canvas.Image

	c fyne.CanvasObject
}

func NewNowPlayingCard() *NowPlayingCard {
	n := &NowPlayingCard{
		trackName:  widget.NewLabel(""),
		artistName: widget.NewLabel(""),
		albumName:  widget.NewLabel(""),
		cover:      &canvas.Image{},
	}
	n.ExtendBaseWidget(n)
	n.trackName.Wrapping = fyne.TextTruncate
	n.trackName.TextStyle = fyne.TextStyle{Bold: true}
	n.artistName.Wrapping = fyne.TextTruncate
	n.albumName.Wrapping = fyne.TextTruncate
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
	n.artistName.Text = artist
	n.albumName.Text = album
	n.cover.Image = cover
	n.c.Refresh()
}
