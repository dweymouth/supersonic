package theme

import (
	"bytes"
	"errors"
	"image/color"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	ColorNameListHeader     fyne.ThemeColorName = "ListHeader"
	ColorNamePageBackground fyne.ThemeColorName = "PageBackground"
	ColorNamePageHeader     fyne.ThemeColorName = "PageHeader"
)

var (
	AlbumIcon       fyne.Resource = theme.NewThemedResource(res.ResDiscSvg)
	ArtistIcon      fyne.Resource = theme.NewThemedResource(res.ResPeopleSvg)
	FavoriteIcon    fyne.Resource = theme.NewThemedResource(res.ResHeartFilledSvg)
	NotFavoriteIcon fyne.Resource = theme.NewThemedResource(res.ResHeartOutlineSvg)
	NowPlayingIcon  fyne.Resource = theme.NewThemedResource(res.ResHeadphonesSvg)
	PlaylistIcon    fyne.Resource = theme.NewThemedResource(res.ResPlaylistSvg)
	ShuffleIcon     fyne.Resource = theme.NewThemedResource(res.ResShuffleSvg)
	TracksIcon      fyne.Resource = theme.NewThemedResource(res.ResMusicnotesSvg)
	GenreIcon       fyne.Resource = theme.NewThemedResource(res.ResTheatermasksSvg)
	FilterIcon      fyne.Resource = theme.NewThemedResource(res.ResFilterSvg)
	RepeatIcon      fyne.Resource = theme.NewThemedResource(res.ResRepeatSvg)
	RepeatOneIcon   fyne.Resource = theme.NewThemedResource(res.ResRepeatoneSvg)
)

type AppearanceMode string

const (
	AppearanceLight AppearanceMode = "Light"
	AppearanceDark  AppearanceMode = "Dark"
	AppearanceAuto  AppearanceMode = "Auto"

	DefaultAppearance AppearanceMode = AppearanceDark
)

var (
	normalFont fyne.Resource
	boldFont   fyne.Resource
)

type MyTheme struct {
	NormalFont   string
	BoldFont     string
	config       *backend.ThemeConfig
	themeFileDir string

	loadedThemeFilename string
	loadedThemeFile     *ThemeFile
	defaultThemeFile    *ThemeFile
}

var _ fyne.Theme = (*MyTheme)(nil)

func NewMyTheme(config *backend.ThemeConfig, themeFileDir string) *MyTheme {
	m := &MyTheme{config: config, themeFileDir: themeFileDir}
	var err error
	if m.defaultThemeFile, err = DecodeThemeFile(bytes.NewReader(res.ResDefaultToml.StaticContent)); err != nil {
		log.Fatalf("Failed to load builtin theme: %v", err.Error())
	}
	return m
}

func (m *MyTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	// load theme file if necessary
	if m.loadedThemeFile == nil || m.config.ThemeFile != m.loadedThemeFilename {
		t, err := ReadThemeFile(path.Join(m.themeFileDir, m.config.ThemeFile))
		if err == nil {
			m.loadedThemeFile = t
		} else {
			log.Printf("failed to load theme file %q: %s", m.config.ThemeFile, err.Error())
			m.loadedThemeFile = m.defaultThemeFile
		}
		m.loadedThemeFilename = m.config.ThemeFile
	}

	variant := m.getVariant()
	thFile := m.loadedThemeFile
	if !thFile.SupportsVariant(variant) {
		thFile = m.defaultThemeFile
	}
	colors := thFile.DarkColors
	defColors := m.defaultThemeFile.DarkColors
	if variant == theme.VariantLight {
		colors = thFile.LightColors
		defColors = m.defaultThemeFile.LightColors
	}
	switch name {
	case ColorNameListHeader:
		return colorOrDefault(colors.ListHeader, defColors.ListHeader, name, variant)
	case ColorNamePageBackground:
		return colorOrDefault(colors.PageBackground, defColors.PageBackground, name, variant)
	case ColorNamePageHeader:
		return colorOrDefault(colors.PageHeader, defColors.PageHeader, name, variant)
	case theme.ColorNameBackground:
		return colorOrDefault(colors.Background, defColors.Background, name, variant)
	case theme.ColorNameButton:
		return colorOrDefault(colors.Button, defColors.Button, name, variant)
	case theme.ColorNameDisabled:
		return colorOrDefault(colors.Disabled, defColors.Disabled, name, variant)
	case theme.ColorNameDisabledButton:
		return colorOrDefault(colors.DisabledButton, defColors.DisabledButton, name, variant)
	case theme.ColorNameError:
		return colorOrDefault(colors.Error, defColors.Error, name, variant)
	case theme.ColorNameFocus:
		return colorOrDefault(colors.Focus, defColors.Focus, name, variant)
	case theme.ColorNameForeground:
		return colorOrDefault(colors.Foreground, defColors.Foreground, name, variant)
	case theme.ColorNameHover:
		return colorOrDefault(colors.Hover, defColors.Hover, name, variant)
	case theme.ColorNameHyperlink:
		return colorOrDefault(colors.Hyperlink, defColors.Hyperlink, name, variant)
	case theme.ColorNameInputBackground:
		return colorOrDefault(colors.InputBackground, defColors.InputBackground, name, variant)
	case theme.ColorNameInputBorder:
		return colorOrDefault(colors.InputBorder, defColors.InputBorder, name, variant)
	case theme.ColorNameMenuBackground:
		return colorOrDefault(colors.MenuBackground, defColors.MenuBackground, name, variant)
	case theme.ColorNameOverlayBackground:
		return colorOrDefault(colors.OverlayBackground, defColors.OverlayBackground, name, variant)
	case theme.ColorNamePlaceHolder:
		return colorOrDefault(colors.Placeholder, defColors.Placeholder, name, variant)
	case theme.ColorNamePressed:
		return colorOrDefault(colors.Pressed, defColors.Pressed, name, variant)
	case theme.ColorNamePrimary:
		return colorOrDefault(colors.Primary, defColors.Primary, name, variant)
	case theme.ColorNameScrollBar:
		return colorOrDefault(colors.ScrollBar, defColors.ScrollBar, name, variant)
	case theme.ColorNameSelection:
		return colorOrDefault(colors.Selection, defColors.Selection, name, variant)
	case theme.ColorNameSeparator:
		return colorOrDefault(colors.Separator, defColors.Separator, name, variant)
	case theme.ColorNameShadow:
		return colorOrDefault(colors.Shadow, defColors.Shadow, name, variant)
	case theme.ColorNameSuccess:
		return colorOrDefault(colors.Success, defColors.Success, name, variant)
	case theme.ColorNameWarning:
		return colorOrDefault(colors.Warning, defColors.Warning, name, variant)
	default:
		return colorOrDefault("", "", name, variant)
	}
}

func colorOrDefault(colorStr, defColorStr string, name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if c, err := ColorStringToColor(colorStr); err == nil {
		return c
	}
	if c, err := ColorStringToColor(defColorStr); err == nil {
		return c
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m *MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Returns a map [themeFileName] -> displayName
func (m *MyTheme) ListThemeFiles() map[string]string {
	files, _ := filepath.Glob(m.themeFileDir + "/*.toml")
	result := make(map[string]string)
	for _, filepath := range files {
		if themeFile, err := ReadThemeFile(filepath); err == nil {
			result[path.Base(filepath)] = themeFile.SupersonicTheme.Name
		}
	}
	return result
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

func (m *MyTheme) getVariant() fyne.ThemeVariant {
	v := DefaultAppearance // default if config has invalid or missing setting
	if sharedutil.SliceContains(
		[]string{string(AppearanceLight), string(AppearanceDark), string(AppearanceAuto)},
		m.config.Appearance) {
		v = AppearanceMode(m.config.Appearance)
	}

	if AppearanceMode(v) == AppearanceDark {
		return theme.VariantDark
	} else if AppearanceMode(v) == AppearanceLight {
		return theme.VariantLight
	}
	return fyne.CurrentApp().Settings().ThemeVariant()
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
