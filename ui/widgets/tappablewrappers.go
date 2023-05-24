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

	NoPointerCursor bool
	OnTapped        func()
}

func NewTappbaleIcon(res fyne.Resource) *TappableIcon {
	icon := &TappableIcon{}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)

	return icon
}

var _ fyne.Tappable = (*TappableIcon)(nil)

func (t *TappableIcon) Tapped(_ *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

var _ desktop.Hoverable = (*TappableIcon)(nil)

func (t *TappableIcon) MouseIn(*desktop.MouseEvent) {}

func (t *TappableIcon) MouseOut() {}

func (t *TappableIcon) MouseMoved(*desktop.MouseEvent) {}

func (t *TappableIcon) Cursor() desktop.Cursor {
	if t.NoPointerCursor {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

// TappableImage is a tappable wrapper of canvas.Image
type TappableImage struct {
	widget.BaseWidget
	canvas.Image

	DisableTapping    bool
	OnTapped          func(*fyne.PointEvent)
	OnTappedSecondary func(*fyne.PointEvent)
}

func NewTappableImage(onTapped func(*fyne.PointEvent)) *TappableImage {
	t := &TappableImage{OnTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *TappableImage) Hide() {
	t.BaseWidget.Hide()
}

func (t *TappableImage) Show() {
	t.BaseWidget.Show()
}

func (t *TappableImage) Move(pos fyne.Position) {
	t.BaseWidget.Move(pos)
}

func (t *TappableImage) Refresh() {
	t.BaseWidget.Refresh()
}

func (t *TappableImage) Resize(size fyne.Size) {
	t.BaseWidget.Resize(size)
}

func (t *TappableImage) Cursor() desktop.Cursor {
	if !t.DisableTapping && t.haveImage() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (t *TappableImage) Tapped(e *fyne.PointEvent) {
	if !t.DisableTapping && t.haveImage() && t.OnTapped != nil {
		t.OnTapped(e)
	}
}

func (t *TappableImage) TappedSecondary(e *fyne.PointEvent) {
	if !t.DisableTapping && t.haveImage() && t.OnTappedSecondary != nil {
		t.OnTappedSecondary(e)
	}
}

func (t *TappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(&t.Image)
}

func (t *TappableImage) haveImage() bool {
	return t.Image.Resource != nil || t.Image.Image != nil
}
