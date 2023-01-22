package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type TappableImage struct {
	widget.BaseWidget
	canvas.Image

	OnTapped func()
}

func NewTappableImage() *TappableImage {
	t := &TappableImage{}
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
