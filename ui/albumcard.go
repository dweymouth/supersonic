package ui

import (
	"context"
	"image"
	"log"

	"supersonic/ui/layout"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

var _ fyne.Widget = (*AlbumCard)(nil)

var _ fyne.Tappable = (*TappableImage)(nil)

type TappableImage struct {
	im *canvas.Image

	onTapped func()
}

func (t *TappableImage) Tapped(*fyne.PointEvent) {
	log.Println("tapped!")
	t.onTapped()
}

func (t *TappableImage) SetImage(im image.Image) {
	t.im.Image = im
	t.im.Refresh()
}

type AlbumCard struct {
	widget.BaseWidget

	albumID   string
	title     *widget.Label
	artist    *widget.Label
	container *fyne.Container

	// updated by AlbumGrid
	Cover *TappableImage

	// these fields are used by AlbumGrid to track async update tasks
	PrevAlbumID   string
	ImgLoadCancel context.CancelFunc

	OnTapped func()
}

func NewAlbumCard() *AlbumCard {
	a := &AlbumCard{
		title:  widget.NewLabel(""),
		artist: widget.NewLabel(""),
		Cover: &TappableImage{
			im: &canvas.Image{},
		},
	}
	a.ExtendBaseWidget(a)
	a.title.Wrapping = fyne.TextTruncate
	a.artist.Wrapping = fyne.TextTruncate
	a.title.TextStyle = fyne.TextStyle{Bold: true}
	a.Cover.im.SetMinSize(fyne.NewSize(200, 200))
	a.Cover.im.FillMode = canvas.ImageFillContain

	a.createContainer()
	return a
}

func (a *AlbumCard) createContainer() {
	titleArtist := container.New(&layout.VboxCustomPadding{ExtraPad: -16}, a.title, a.artist)
	c := container.New(&layout.VboxCustomPadding{ExtraPad: -5}, a.Cover.im, titleArtist)
	pad := &layout.CenterPadLayout{PadLeftRight: 20, PadTopBottom: 10}
	a.container = container.New(pad, c)
}

func (a *AlbumCard) Update(al *subsonic.AlbumID3) {
	a.title.SetText(al.Name)
	a.artist.SetText(al.Artist)
	a.albumID = al.ID
}

func (a *AlbumCard) AlbumID() string {
	return a.albumID
}

func (a *AlbumCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
	//return widget.NewSimpleRenderer(container.New(&layout.VboxCustomPadding{-10}, a.Cover.im, a.title, a.artist))
}

func (a *AlbumCard) Tapped(*fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}
