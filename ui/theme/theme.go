package theme

import (
	"errors"
	"image/color"
	"io/ioutil"
	"log"
	"strings"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/sharedutil"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const ColorNamePageBackground fyne.ThemeColorName = "PageBackground"

const (
	IconNameNowPlaying  fyne.ThemeIconName = "NowPlaying"
	IconNameFavorite    fyne.ThemeIconName = "Favorite"
	IconNameNotFavorite fyne.ThemeIconName = "NotFavorite"
	IconNameAlbum       fyne.ThemeIconName = "Album"
	IconNameArtist      fyne.ThemeIconName = "Artist"
	IconNameGenre       fyne.ThemeIconName = "Genre"
	IconNamePlaylist    fyne.ThemeIconName = "Playlist"
	IconNameShuffle     fyne.ThemeIconName = "Shuffle"
)

type AppearanceMode string

const (
	AppearanceLight AppearanceMode = "Light"
	AppearanceDark  AppearanceMode = "Dark"
	AppearanceAuto  AppearanceMode = "Auto"
)

var (
	normalFont fyne.Resource
	boldFont   fyne.Resource
)

type MyTheme struct {
	NormalFont  string
	BoldFont    string
	config *backend.ThemeConfig
}

var _ fyne.Theme = (*MyTheme)(nil)

func NewMyTheme(config *backend.ThemeConfig) *MyTheme {
	m := &MyTheme{config: config}
	m.createThemeIcons()
	return m
}

func (m *MyTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	variant := m.getVariant()
	switch name {
	case ColorNamePageBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 15, G: 15, B: 15, A: 255}
		}
		return color.RGBA{R: 250, G: 250, B: 250, A: 255}
	case theme.ColorNameBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 35, G: 35, B: 35, A: 255}
		}
		return color.RGBA{R: 225, G: 223, B: 225, A: 255}
	case theme.ColorNameScrollBar:
		if variant == theme.VariantDark {
			return theme.DarkTheme().Color(theme.ColorNameForeground, variant)
		}
		return theme.LightTheme().Color(theme.ColorNameForeground, variant)
	case theme.ColorNameButton:
		if variant == theme.VariantDark {
			return color.RGBA{R: 20, G: 20, B: 20, A: 50}
		}
		return color.RGBA{R: 200, G: 200, B: 200, A: 240}
	case theme.ColorNameInputBackground:
		if variant == theme.VariantDark {
			return color.RGBA{R: 20, G: 20, B: 20, A: 50}
		}
	case theme.ColorNameForeground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 10, G: 10, B: 10, A: 255}
		}
	case theme.ColorNamePrimary:
		if variant == theme.VariantLight {
			return color.RGBA{R: 25, G: 25, B: 250, A: 255}
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m *MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	variant := m.getVariant()
	switch name {
	case IconNameAlbum:
		if variant == theme.VariantDark {
			return res.ResDiscInvertPng
		}
		return res.ResDiscPng
	case IconNameArtist:
		if variant == theme.VariantDark {
			return res.ResPeopleInvertPng
		}
		return res.ResPeoplePng
	case IconNameFavorite:
		if variant == theme.VariantDark {
			return res.ResHeartFilledInvertPng
		}
		return res.ResHeartFilledPng
	case IconNameNotFavorite:
		if variant == theme.VariantDark {
			return res.ResHeartOutlineInvertPng
		}
		return res.ResHeartOutlinePng
	case IconNameGenre:
		if variant == theme.VariantDark {
			return res.ResTheatermasksInvertPng
		}
		return res.ResTheatermasksPng
	case IconNameNowPlaying:
		if variant == theme.VariantDark {
			return res.ResHeadphonesInvertPng
		}
		return res.ResHeadphonesPng
	case IconNamePlaylist:
		if variant == theme.VariantDark {
			return res.ResPlaylistInvertPng
		}
		return res.ResPlaylistPng
	case IconNameShuffle:
		if variant == theme.VariantDark {
			return res.ResShuffleInvertSvg
		}
		return res.ResShuffleSvg
	default:
		return theme.DefaultTheme().Icon(name)
	}
}

type myThemedResource struct {
	myTheme      MyTheme
	darkVariant  *fyne.StaticResource
	lightVariant *fyne.StaticResource
}

var _ fyne.Resource = myThemedResource{}

func (p myThemedResource) Content() []byte {
	if p.myTheme.getVariant() == theme.VariantDark {
		return p.darkVariant.StaticContent
	}
	return p.lightVariant.StaticContent
}

func (p myThemedResource) Name() string {
	if p.myTheme.getVariant() == theme.VariantDark {
		return p.darkVariant.StaticName
	}
	return p.lightVariant.StaticName
}

var (
	AlbumIcon       fyne.Resource
	ArtistIcon      fyne.Resource
	FavoriteIcon    fyne.Resource
	NotFavoriteIcon fyne.Resource
	GenreIcon       fyne.Resource
	NowPlayingIcon  fyne.Resource
	PlaylistIcon    fyne.Resource
	ShuffleIcon     fyne.Resource
)

// MUST be called at startup!
func (m MyTheme) createThemeIcons() {
	AlbumIcon = myThemedResource{myTheme: m, darkVariant: res.ResDiscInvertPng, lightVariant: res.ResDiscPng}
	ArtistIcon = myThemedResource{myTheme: m, darkVariant: res.ResPeopleInvertPng, lightVariant: res.ResPeoplePng}
	FavoriteIcon = myThemedResource{myTheme: m, darkVariant: res.ResHeartFilledInvertPng, lightVariant: res.ResHeartFilledPng}
	NotFavoriteIcon = myThemedResource{myTheme: m, darkVariant: res.ResHeartOutlineInvertPng, lightVariant: res.ResHeartOutlinePng}
	GenreIcon = myThemedResource{myTheme: m, darkVariant: res.ResTheatermasksInvertPng, lightVariant: res.ResTheatermasksPng}
	NowPlayingIcon = myThemedResource{myTheme: m, darkVariant: res.ResHeadphonesInvertPng, lightVariant: res.ResHeadphonesPng}
	PlaylistIcon = myThemedResource{myTheme: m, darkVariant: res.ResPlaylistInvertPng, lightVariant: res.ResPlaylistPng}
	ShuffleIcon = myThemedResource{myTheme: m, darkVariant: res.ResShuffleInvertSvg, lightVariant: res.ResShuffleSvg}
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
	v := "Dark" // default if config has invalid or missing setting
	if sharedutil.SliceContains(
		[]string{string(AppearanceLight), string(AppearanceDark), string(AppearanceAuto)},
		m.config.Appearance) {
		v = m.config.Appearance
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
