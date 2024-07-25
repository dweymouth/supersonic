package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
)

type colorOverrideTheme struct {
	colorTransformName fyne.ThemeColorName
	colorTransform     func(color.Color) color.Color
}

func WithColorTransformOverride(name fyne.ThemeColorName, transform func(color.Color) color.Color) fyne.Theme {
	return &colorOverrideTheme{
		colorTransformName: name,
		colorTransform:     transform,
	}
}

var _ fyne.Theme = (*colorOverrideTheme)(nil)

func (c *colorOverrideTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	col := fyne.CurrentApp().Settings().Theme().Color(n, v)
	if n == c.colorTransformName {
		return c.colorTransform(col)
	}
	return col
}

func (*colorOverrideTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(s)
}

func (*colorOverrideTheme) Icon(s fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(s)
}

func (*colorOverrideTheme) Size(s fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(s)
}
