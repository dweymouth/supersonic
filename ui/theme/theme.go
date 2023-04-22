package theme

import (
	"errors"
	"image/color"
	"io/ioutil"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const ColorNamePageBackground fyne.ThemeColorName = "PageBackground"

var (
	normalFont fyne.Resource
	boldFont   fyne.Resource
)

type MyTheme struct {
	NormalFont string
	BoldFont   string
}

var _ fyne.Theme = (*MyTheme)(nil)

func (m *MyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
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

func (m *MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *MyTheme) Font(style fyne.TextStyle) fyne.Resource {
	switch style {
	case fyne.TextStyle{}:
		if m.NormalFont != "" && normalFont == nil {
			if content, err := readTTFFile(m.NormalFont); err != nil {
				m.NormalFont = ""
				m.BoldFont = ""
			} else {
				normalFont = fyne.NewStaticResource("normalFont", content)
			}
		}
		if normalFont != nil {
			return normalFont
		}
	case fyne.TextStyle{Bold: true}:
		if m.BoldFont != "" && boldFont == nil {
			if content, err := ioutil.ReadFile(m.BoldFont); err != nil {
				m.BoldFont = ""
			} else {
				normalFont = fyne.NewStaticResource("boldFont", content)
			}
		}
		if boldFont != nil {
			return boldFont
		}
		if normalFont != nil {
			return normalFont
		}
	}

	return theme.DefaultTheme().Font(style)
}

func (m *MyTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func readTTFFile(filepath string) ([]byte, error) {
	if !strings.HasSuffix(filepath, ".ttf") {
		err := errors.New("only .ttf fonts are supported")
		log.Printf("error loading custom font %q: %s", filepath, err.Error())
		return nil, err
	}
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("error loading custom font %q: %s", filepath, err.Error())
	}
	return content, err
}
