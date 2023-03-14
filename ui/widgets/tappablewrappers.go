package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TappableIcon is a tappable wrapper of widget.Icon
type TappableIcon struct {
	widget.Icon

	OnTapped func()
}

func NewTappbaleIcon(res fyne.Resource) *TappableIcon {
	icon := &TappableIcon{}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)

	return icon
}

func (t *TappableIcon) Tapped(_ *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

func (t *TappableIcon) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

// TappableImage is a tappable wrapper of canvas.Image
type TappableImage struct {
	widget.BaseWidget
	canvas.Image

	OnTapped func()
}

func NewTappableImage(onTapped func()) *TappableImage {
	t := &TappableImage{OnTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *TappableImage) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *TappableImage) Tapped(e *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *TappableImage) Refresh() {
	t.Image.Refresh()
	t.BaseWidget.Refresh()
}

func (t *TappableImage) Resize(size fyne.Size) {
	t.Image.Resize(size)
	t.BaseWidget.Resize(size)
}

func (t *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(&t.Image)
}
