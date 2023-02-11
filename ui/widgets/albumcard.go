package widgets

import (
	"context"
	"image"
	"strconv"

	"supersonic/res"
	"supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/dweymouth/go-subsonic/subsonic"
)

var _ fyne.Widget = (*AlbumCard)(nil)

var _ fyne.Widget = (*albumCover)(nil)
var _ fyne.Tappable = (*albumCover)(nil)

type albumCover struct {
	widget.BaseWidget

	Im             *canvas.Image
	playbtn        *canvas.Image
	OnDoubleTapped func()
	OnTapped       func()
}

func newAlbumCover() *albumCover {
	a := &albumCover{}
	a.ExtendBaseWidget(a)
	a.Im = &canvas.Image{FillMode: canvas.ImageFillContain}
	a.Im.SetMinSize(fyne.NewSize(200, 200))
	a.playbtn = &canvas.Image{FillMode: canvas.ImageFillContain, Resource: res.ResPlaybuttonPng}
	a.playbtn.SetMinSize(fyne.NewSize(60, 60))
	a.playbtn.Hidden = true
	return a
}

func (a *albumCover) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.NewMax(a.Im, container.NewCenter(a.playbtn)),
	)
}

func (a *albumCover) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (a *albumCover) Tapped(e *fyne.PointEvent) {
	if isInside(a.center(), a.playbtn.Size().Height/2, e.Position) {
		if a.OnDoubleTapped != nil {
			a.OnDoubleTapped()
		}
		return
	}
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *albumCover) MouseIn(*desktop.MouseEvent) {
	a.playbtn.Hidden = false
	a.Refresh()
}

func (a *albumCover) MouseOut() {
	a.playbtn.Hidden = true
	a.Refresh()
}

// TODO: figure out why circle around play button isn't being displayed
func (a *albumCover) MouseMoved(e *desktop.MouseEvent) {
	if isInside(a.center(), a.playbtn.MinSize().Height/2, e.Position) {
		a.playbtn.SetMinSize(fyne.NewSize(65, 65))
	} else {
		a.playbtn.SetMinSize(fyne.NewSize(60, 60))
	}
	a.Refresh()
}

func (a *albumCover) center() fyne.Position {
	return fyne.NewPos(a.Size().Width/2, a.Size().Height/2)
}

func (a *albumCover) SetImage(im image.Image) {
	a.Im.Image = im
	a.Refresh()
}

func isInside(origin fyne.Position, radius float32, point fyne.Position) bool {
	x, y := (point.X - origin.X), (point.Y - origin.Y)
	return x*x+y*y <= radius*radius
}

type AlbumCard struct {
	widget.BaseWidget

	albumID   string
	artistID  string
	title     *CustomHyperlink
	artist    *CustomHyperlink
	year      *widget.Label
	container *fyne.Container

	showYear bool

	// updated by AlbumGrid
	Cover *albumCover

	// these fields are used by AlbumGrid to track async update tasks
	PrevAlbumID   string
	ImgLoadCancel context.CancelFunc

	OnPlay           func()
	OnShowAlbumPage  func()
	OnShowArtistPage func()
}

func NewAlbumCard(showYear bool) *AlbumCard {
	a := &AlbumCard{
		title:    NewCustomHyperlink(),
		artist:   NewCustomHyperlink(),
		year:     widget.NewLabel(""),
		Cover:    newAlbumCover(),
		showYear: showYear,
	}
	a.ExtendBaseWidget(a)
	a.Cover.OnDoubleTapped = func() {
		if a.OnPlay != nil {
			a.OnPlay()
		}
	}
	showAlbumFn := func() {
		if a.OnShowAlbumPage != nil {
			a.OnShowAlbumPage()
		}
	}
	a.Cover.OnTapped = showAlbumFn
	a.title.OnTapped = showAlbumFn
	a.artist.OnTapped = func() {
		if a.OnShowArtistPage != nil {
			a.OnShowArtistPage()
		}
	}

	a.createContainer()
	return a
}

func (a *AlbumCard) createContainer() {
	var secondLabel fyne.Widget = a.artist
	if a.showYear {
		secondLabel = a.year
	}
	info := container.New(&layouts.VboxCustomPadding{ExtraPad: -16}, a.title, secondLabel)
	c := container.New(&layouts.VboxCustomPadding{ExtraPad: -5}, a.Cover, info)
	pad := &layouts.CenterPadLayout{PadLeftRight: 20, PadTopBottom: 10}
	a.container = container.New(pad, c)
}

func (a *AlbumCard) Update(al *subsonic.AlbumID3) {
	a.title.SetText(al.Name)
	a.artist.SetText(al.Artist)
	a.year.SetText(strconv.Itoa(al.Year))
	a.albumID = al.ID
	a.artistID = al.ArtistID
	a.Cover.playbtn.Hidden = true
}

func (a *AlbumCard) AlbumID() string {
	return a.albumID
}

func (a *AlbumCard) ArtistID() string {
	return a.artistID
}

func (a *AlbumCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
