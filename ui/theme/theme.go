package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const ColorNamePageBackground fyne.ThemeColorName = "PageBackground"

type MyTheme struct{}

var _ fyne.Theme = (*MyTheme)(nil)

func (m MyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case ColorNamePageBackground:
		return color.RGBA{R: 15, G: 15, B: 15, A: 255}
	case theme.ColorNameBackground:
		return color.RGBA{R: 30, G: 30, B: 30, A: 255}
	case theme.ColorNameScrollBar:
		return theme.DarkTheme().Color(theme.ColorNameForeground, variant)
	case theme.ColorNameButton:
		return color.RGBA{R: 20, G: 20, B: 20, A: 50}
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 20, G: 20, B: 20, A: 50}
	}
	return theme.DarkTheme().Color(name, variant)
}

func (m MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m MyTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m MyTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
