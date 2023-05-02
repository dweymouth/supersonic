package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type ThemedRectangle struct {
	widget.BaseWidget

	rect *canvas.Rectangle

	ColorName fyne.ThemeColorName
}

func NewThemedRectangle(colorName fyne.ThemeColorName) *ThemedRectangle {
	t := &ThemedRectangle{
		ColorName: colorName,
		rect: canvas.NewRectangle(fyne.CurrentApp().Settings().Theme().Color(colorName,
			fyne.CurrentApp().Settings().ThemeVariant())),
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ThemedRectangle) Refresh() {
	t.rect.FillColor = fyne.CurrentApp().Settings().Theme().Color(t.ColorName,
		fyne.CurrentApp().Settings().ThemeVariant())
	t.BaseWidget.Refresh()
}

func (t *ThemedRectangle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.rect)
}
