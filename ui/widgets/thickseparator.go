package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ThickSeparator struct {
	widget.BaseWidget
	line canvas.Line
}

func NewThickSeparator() *ThickSeparator {
	t := &ThickSeparator{
		line: canvas.Line{
			StrokeWidth: 3,
			StrokeColor: theme.DisabledColor(),
		},
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ThickSeparator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(&t.line)
}
