package widgets

import (
	"context"
	"image"

	"supersonic/ui/layout"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/dweymouth/go-subsonic"
)

var _ fyne.Widget = (*AlbumCard)(nil)

var _ fyne.Widget = (*albumCover)(nil)
var _ fyne.Tappable = (*albumCover)(nil)
var _ fyne.DoubleTappable = (*albumCover)(nil)

type albumCover struct {
	widget.BaseWidget

	Im             *canvas.Image
	OnDoubleTapped func()
	OnTapped       func()
}

func newAlbumCover() *albumCover {
	a := &albumCover{}
	a.ExtendBaseWidget(a)
	a.Im = &canvas.Image{FillMode: canvas.ImageFillContain}
	a.Im.SetMinSize(fyne.NewSize(200, 200))
	return a
}

func (a *albumCover) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.Im)
}

func (a *albumCover) DoubleTapped(e *fyne.PointEvent) {
	if a.OnDoubleTapped != nil {
		a.OnDoubleTapped()
	}
}

func (a *albumCover) Tapped(e *fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *albumCover) SetImage(im image.Image) {
	a.Im.Image = im
	a.Refresh()
}

type AlbumCard struct {
	widget.BaseWidget

	albumID   string
	title     *widget.Label
	artist    *widget.Label
	container *fyne.Container

	// updated by AlbumGrid
	Cover *albumCover

	// these fields are used by AlbumGrid to track async update tasks
	PrevAlbumID   string
	ImgLoadCancel context.CancelFunc

	OnPlay func()
}

func (a *AlbumCard) MouseIn(*desktop.MouseEvent) {}

func (a *AlbumCard) MouseOut() {}

func (a *AlbumCard) MouseMoved(*desktop.MouseEvent) {}

func NewAlbumCard() *AlbumCard {
	a := &AlbumCard{
		title:  widget.NewLabel(""),
		artist: widget.NewLabel(""),
		Cover:  newAlbumCover(),
	}
	a.ExtendBaseWidget(a)
	a.Cover.OnDoubleTapped = func() {
		if a.OnPlay != nil {
			a.OnPlay()
		}
	}
	a.title.Wrapping = fyne.TextTruncate
	a.artist.Wrapping = fyne.TextTruncate
	a.title.TextStyle = fyne.TextStyle{Bold: true}

	a.createContainer()
	return a
}

func (a *AlbumCard) createContainer() {
	titleArtist := container.New(&layout.VboxCustomPadding{ExtraPad: -16}, a.title, a.artist)
	c := container.New(&layout.VboxCustomPadding{ExtraPad: -5}, a.Cover, titleArtist)
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
}
