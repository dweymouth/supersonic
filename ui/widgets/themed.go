package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type ThemedIconButton struct {
	widget.Button

	IconName fyne.ThemeIconName
}

func NewThemedIconButton(iconName fyne.ThemeIconName, text string, action func()) *ThemedIconButton {
	b := &ThemedIconButton{
		IconName: iconName,
		Button: widget.Button{
			Text:     text,
			OnTapped: action,
		},
	}
	b.updateIcon()
	b.ExtendBaseWidget(b)
	return b
}

func (b *ThemedIconButton) updateIcon() {
	b.Icon = fyne.CurrentApp().Settings().Theme().Icon(b.IconName)
}

func (b *ThemedIconButton) Refresh() {
	b.updateIcon()
	b.Button.Refresh()
}

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
