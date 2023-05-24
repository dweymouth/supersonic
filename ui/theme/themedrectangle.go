package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type ThemedRectangle struct {
	widget.BaseWidget

	rect *canvas.Rectangle

	ColorName       fyne.ThemeColorName
	BorderWidth     float32
	BorderColorName fyne.ThemeColorName
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
	settings := fyne.CurrentApp().Settings()
	theme := settings.Theme()
	t.rect.FillColor = theme.Color(t.ColorName, settings.ThemeVariant())
	t.rect.StrokeWidth = t.BorderWidth
	t.rect.StrokeColor = theme.Color(t.BorderColorName, settings.ThemeVariant())
	t.BaseWidget.Refresh()
}

func (t *ThemedRectangle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.rect)
}
