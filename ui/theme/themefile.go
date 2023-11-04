package theme

import (
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/pelletier/go-toml/v2"
)

var validThemeVersions = []string{"0.1", "0.2"}

type ThemeFileHeader struct {
	Name          string
	Version       string
	SupportsDark  bool
	SupportsLight bool
}

type ThemeFile struct {
	SupersonicTheme ThemeFileHeader

	DarkColors  ThemeColors
	LightColors ThemeColors
}

type ThemeColors struct {
	// Supersonic-specific colors

	ListHeader string

	PageBackground string

	// Fyne colors

	Background string

	Button string

	DisabledButton string

	Disabled string

	Error string

	Focus string

	Foreground string

	Hover string

	// since Supersonic theme file version 0.2 (Fyne version 2.4)
	Hyperlink string

	InputBackground string

	InputBorder string

	MenuBackground string

	OverlayBackground string

	Placeholder string

	Pressed string

	Primary string

	ScrollBar string

	Selection string

	Separator string

	Shadow string

	Success string

	Warning string
}

func ReadThemeFile(filePath string) (*ThemeFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return DecodeThemeFile(f)
}

func DecodeThemeFile(reader io.Reader) (*ThemeFile, error) {
	theme := &ThemeFile{}
	if err := toml.NewDecoder(reader).Decode(theme); err != nil {
		return nil, err
	}

	if theme.SupersonicTheme.Name == "" || !sharedutil.SliceContains(validThemeVersions, theme.SupersonicTheme.Version) {
		return nil, errors.New("invalid theme file name or version")
	}
	if !(theme.SupersonicTheme.SupportsDark || theme.SupersonicTheme.SupportsLight) {
		return nil, errors.New("invalid theme file: must support one or both of light/dark")
	}
	updateThemeToLatestVersion(theme)

	return theme, nil
}

func (t *ThemeFile) SupportsVariant(v fyne.ThemeVariant) bool {
	if v == theme.VariantDark {
		return t.SupersonicTheme.SupportsDark
	}
	return t.SupersonicTheme.SupportsLight
}

// Parses a CSS-style #RRGGBB or #RRGGBBAA string
func ColorStringToColor(colorStr string) (color.Color, error) {
	if !strings.HasPrefix(colorStr, "#") || !sharedutil.SliceContains([]int{7, 9}, len(colorStr)) {
		return color.Black, errors.New("invalid color string")
	}
	colorBytes := make([]byte, 4)
	n, err := hex.Decode(colorBytes, []byte(colorStr[1:]))
	if err != nil {
		return color.Black, fmt.Errorf("invalid color string: %s", err.Error())
	}
	if n == 3 {
		colorBytes[3] = 255 // opaque alpha
	}
	return color.RGBA{R: colorBytes[0], G: colorBytes[1], B: colorBytes[2], A: colorBytes[3]}, nil
}

func updateThemeToLatestVersion(themeFile *ThemeFile) {
	switch themeFile.SupersonicTheme.Version {
	case "0.1":
		// in version 0.1, hyperlinks were drawn with the Primary color
		themeFile.DarkColors.Hyperlink = themeFile.DarkColors.Primary
		themeFile.LightColors.Hyperlink = themeFile.LightColors.Primary
	}
}
