package theme

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"math"
	"strings"
)

// SliderRanges defines the min/max ranges for saturation and contrast sliders per base mode
type SliderRanges struct {
	SatMin, SatMax           float64
	ContrastMin, ContrastMax float64
}

// GetSliderRanges returns the clamp ranges for saturation and contrast based on base mode
// This is the single source of truth used by both UI sliders and palette generation
func GetSliderRanges(baseMode string) SliderRanges {
	switch strings.ToLower(baseMode) {
	case "light":
		return SliderRanges{SatMin: 0.0, SatMax: 1.0, ContrastMin: 0.0, ContrastMax: 0.5}
	case "black":
		return SliderRanges{SatMin: 0.5, SatMax: 1.0, ContrastMin: 0.5, ContrastMax: 1.0}
	case "grey":
		return SliderRanges{SatMin: 0.0, SatMax: 1.0, ContrastMin: 0.5, ContrastMax: 1.0}
	default:
		return SliderRanges{SatMin: 0.0, SatMax: 1.0, ContrastMin: 0.5, ContrastMax: 1.0}
	}
}

// Palette represents a generated color palette from an accent color
type Palette struct {
	Accent         color.Color
	Background     color.Color
	Surface        color.Color
	SurfaceHover   color.Color
	MenuBackground color.Color // Subtle accent tint for menus
	TextPrimary    color.Color
	TextSecondary  color.Color
	TextOnAccent   color.Color
	Success        color.Color
	Danger         color.Color
	PageBackground color.Color
	PageHeader     color.Color
	ListHeader     color.Color
	Hyperlink      color.Color
}

// GeneratePalette creates a color palette from the given configuration
// All background colors are derived from the accent color with HSL adjustments
func GeneratePalette(accentHex string, saturation, contrast float64, baseMode string) (*Palette, error) {
	accent, err := hexToColor(accentHex)
	if err != nil {
		return nil, fmt.Errorf("invalid accent color: %w", err)
	}

	// Convert accent to HSL for dynamic adjustments
	accentH, accentS, accentL := rgbToHsl(accent)

	// Validate accent color: neon colors (high sat + high lum) are harsh in light mode
	// Log warning but don't block - user can still choose them if they really want
	if accentS > 0.85 && accentL > 0.65 {
		// Neon color detected in light mode - this will be eye-bleeding
		// Silently desaturate the accent slightly to make it usable
		accentS = 0.85
		accent = hslToRgb(accentH, accentS, accentL)
	}

	// Apply safety clamp using the same ranges as UI sliders
	ranges := GetSliderRanges(baseMode)
	saturation = clampFloat(saturation, ranges.SatMin, ranges.SatMax)
	contrast = clampFloat(contrast, ranges.ContrastMin, ranges.ContrastMax)

	// Lightness values calibrated for proper contrast
	// Background < Surface < Text (dark modes) or Background > Surface > Text (light mode)
	var bgL, surfL, pageBgL, pageHeaderL, listHeaderL float64
	var textPrimaryRGB, textSecondaryRGB color.RGBA

	switch strings.ToLower(baseMode) {
	case "black":
		// AMOLED: near-black backgrounds, clear separation
		bgL = 0.04     // Very dark background
		surfL = 0.10   // Slightly lighter surface for cards/inputs
		pageBgL = 0.03 // Page background slightly darker
		pageHeaderL = 0.14
		listHeaderL = 0.12
		// Text: accent hue, very desaturated, high lightness
		textPrimaryRGB = hslToRgb(accentH, 0.06, 0.98)
		textSecondaryRGB = hslToRgb(accentH, 0.05, 0.70)
	case "grey":
		// Grey mode: mid-tone greys with good contrast
		bgL = 0.12     // Dark grey background (darker)
		surfL = 0.18   // Lighter grey for surfaces (darker)
		pageBgL = 0.10 // Slightly darker page bg (darker)
		pageHeaderL = 0.22
		listHeaderL = 0.20
		// Text: accent hue, low saturation
		textPrimaryRGB = hslToRgb(accentH, 0.08, 0.96)
		textSecondaryRGB = hslToRgb(accentH, 0.06, 0.65)
	case "light":
		// Light mode: light backgrounds with DARK text for contrast
		bgL = 0.88     // Light grey background (not pure white for less eye strain)
		surfL = 0.96   // Near-white surface
		pageBgL = 0.92 // Slightly darker page background
		pageHeaderL = 0.82
		listHeaderL = 0.85
		// Text: accent hue, low saturation, dark
		textPrimaryRGB = hslToRgb(accentH, 0.12, 0.10)   // Dark text with subtle tint
		textSecondaryRGB = hslToRgb(accentH, 0.10, 0.35) // Medium grey with tint
	default: // grey (default dark mode)
		bgL = 0.12     // Dark grey background (darker)
		surfL = 0.18   // Lighter grey for surfaces (darker)
		pageBgL = 0.10 // Slightly darker page bg (darker)
		pageHeaderL = 0.22
		listHeaderL = 0.20
		// Text: accent hue, low saturation
		textPrimaryRGB = hslToRgb(accentH, 0.08, 0.96)
		textSecondaryRGB = hslToRgb(accentH, 0.06, 0.65)
	}

	// Apply contrast inversely to backgrounds (darker in dark modes, lighter in light)
	// Contrast 1.0 = neutral, >1.0 = more contrast (deeper bg, brighter text), <1.0 = less contrast
	contrastFactor := contrast
	isLight := baseMode == "light"

	// Adjust lightness: in dark modes, higher contrast = darker bg; in light mode, higher contrast = lighter bg
	adjustLightness := func(l float64) float64 {
		if isLight {
			// Light mode: higher contrast pushes bg lighter
			return clampFloat(l+(contrastFactor-1.0)*0.05, 0.75, 1.0)
		}
		// Dark modes: higher contrast pushes bg darker
		return clampFloat(l-(contrastFactor-1.0)*0.08, 0.0, 0.4)
	}

	// Background saturation: fixed low base with minimal slider influence
	// This prevents backgrounds from becoming oversaturated and losing the accent relationship
	baseBgSat := 0.12 // Fixed low saturation for backgrounds (subtle tint)
	if baseMode == "black" {
		baseBgSat = 0.06 // Black mode: even more subtle
	}
	// Saturation slider has minimal effect on backgrounds (0.5x to 1.5x range)
	bgSatMultiplier := 0.5 + (saturation * 0.5)
	bgSat := baseBgSat * bgSatMultiplier

	bg := hslToRgb(accentH, bgSat, adjustLightness(bgL))
	surf := hslToRgb(accentH, bgSat, adjustLightness(surfL))
	pageBg := hslToRgb(accentH, bgSat, adjustLightness(pageBgL))
	pageHeader := hslToRgb(accentH, bgSat, adjustLightness(pageHeaderL))
	listHeader := hslToRgb(accentH, bgSat, adjustLightness(listHeaderL))

	// Text colors are already set in the switch statement above
	textPrimary := textPrimaryRGB
	textSecondary := textSecondaryRGB

	// Apply saturation to accent (slider affects accent intensity)
	if saturation != 1.0 {
		accent = applySaturationToHSL(accentH, accentS, accentL, saturation)
	}

	// Apply contrast adjustment to accent
	accent = applyContrast(accent, contrast, baseMode == "light")

	// Calculate text color on accent based on luminance
	textOnAccent := calculateTextOnAccent(accent)

	// Generate surface hover color by blending accent with surface
	// Blend ratio is inversely proportional to surface lightness:
	// - Light surfaces (light mode): strong hover (24-28% accent)
	// - Dark surfaces (dark modes): stronger hover (32-36% accent)
	surfLum := (float64(surf.R)*0.299 + float64(surf.G)*0.587 + float64(surf.B)*0.114) / 255.0
	hoverBlendRatio := 0.36 - (surfLum * 0.08) // 0.36 when black, 0.28 when white
	surfaceHover := blendColors(accent, surf, clampFloat(hoverBlendRatio, 0.24, 0.40))

	// Menu background: subtle accent tint, half the intensity of SurfaceHover
	// - Light surfaces: ~12-14% accent
	// - Dark surfaces: ~16-18% accent
	menuBlendRatio := 0.18 - (surfLum * 0.04) // 0.18 when black, 0.14 when white
	menuBackground := blendColors(accent, surf, clampFloat(menuBlendRatio, 0.12, 0.20))

	// Set hyperlink color to accent
	hyperlink := accent

	return &Palette{
		Accent:         accent,
		Background:     bg,
		Surface:        surf,
		SurfaceHover:   surfaceHover,
		MenuBackground: menuBackground,
		TextPrimary:    textPrimary,
		TextSecondary:  textSecondary,
		TextOnAccent:   textOnAccent,
		Success:        color.RGBA{R: 26, G: 179, B: 77, A: 255},
		Danger:         color.RGBA{R: 204, G: 51, B: 51, A: 255},
		PageBackground: pageBg,
		PageHeader:     pageHeader,
		ListHeader:     listHeader,
		Hyperlink:      hyperlink,
	}, nil
}

// hexToColor parses a hex color string (#RRGGBB or #RRGGBBAA)
func hexToColor(hexStr string) (color.RGBA, error) {
	hexStr = strings.TrimSpace(hexStr)
	if !strings.HasPrefix(hexStr, "#") {
		return color.RGBA{}, fmt.Errorf("invalid hex color format")
	}
	hexStr = hexStr[1:]

	var r, g, b, a uint8
	var err error

	switch len(hexStr) {
	case 6:
		r, err = parseHexByte(hexStr[0:2])
		if err != nil {
			return color.RGBA{}, err
		}
		g, err = parseHexByte(hexStr[2:4])
		if err != nil {
			return color.RGBA{}, err
		}
		b, err = parseHexByte(hexStr[4:6])
		if err != nil {
			return color.RGBA{}, err
		}
		a = 255
	case 8:
		r, err = parseHexByte(hexStr[0:2])
		if err != nil {
			return color.RGBA{}, err
		}
		g, err = parseHexByte(hexStr[2:4])
		if err != nil {
			return color.RGBA{}, err
		}
		b, err = parseHexByte(hexStr[4:6])
		if err != nil {
			return color.RGBA{}, err
		}
		a, err = parseHexByte(hexStr[6:8])
		if err != nil {
			return color.RGBA{}, err
		}
	default:
		return color.RGBA{}, fmt.Errorf("invalid hex color length")
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

func parseHexByte(s string) (uint8, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// applyContrast adjusts the contrast of a color
func applyContrast(c color.RGBA, contrast float64, isLightMode bool) color.RGBA {
	var multiplier float64
	if isLightMode {
		multiplier = 0.7 * contrast
	} else {
		multiplier = contrast
	}

	r := clamp(float64(c.R) * multiplier)
	g := clamp(float64(c.G) * multiplier)
	b := clamp(float64(c.B) * multiplier)

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: c.A}
}

// calculateTextOnAccent determines the best text color (black or white) based on luminance
func calculateTextOnAccent(c color.RGBA) color.Color {
	// Calculate luminance using standard formula
	luminance := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)

	if luminance > 128 {
		return color.Black
	}
	return color.White
}

func clamp(v float64) float64 {
	return math.Max(0, math.Min(255, v))
}

// applySaturationToHSL applies a saturation multiplier to HSL values and returns RGB
func applySaturationToHSL(h, s, l, saturationMultiplier float64) color.RGBA {
	newSat := clampFloat(s*saturationMultiplier, 0, 1)
	return hslToRgb(h, newSat, l)
}

// clampFloat constrains a float value between min and max
func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// blendColors blends two colors with the given fraction of the first color
func blendColors(a, b color.Color, fractionA float64) color.Color {
	ra, ga, ba, aa := a.RGBA()
	rb, gb, bb, ab := b.RGBA()

	fractionB := 1 - fractionA
	rAvg := uint8(float64(ra/257)*fractionA + float64(rb/257)*fractionB)
	gAvg := uint8(float64(ga/257)*fractionA + float64(gb/257)*fractionB)
	bAvg := uint8(float64(ba/257)*fractionA + float64(bb/257)*fractionB)
	aAvg := uint8(float64(aa/257)*fractionA + float64(ab/257)*fractionB)
	return color.RGBA{R: rAvg, G: gAvg, B: bAvg, A: aAvg}
}

// rgbToHsl converts RGB to HSL color space
// Returns hue (0-360), saturation (0-1), lightness (0-1)
func rgbToHsl(c color.RGBA) (h, s, l float64) {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	// Lightness
	l = (max + min) / 2

	// Saturation
	if delta == 0 {
		s = 0
		h = 0 // undefined, set to 0
	} else {
		if l < 0.5 {
			s = delta / (max + min)
		} else {
			s = delta / (2 - max - min)
		}

		// Hue
		switch {
		case max == r:
			h = (g - b) / delta
			if g < b {
				h += 6
			}
		case max == g:
			h = (b-r)/delta + 2
		default:
			h = (r-g)/delta + 4
		}
		h *= 60
	}

	return h, s, l
}

// hslToRgb converts HSL to RGB color space
// Takes hue (0-360), saturation (0-1), lightness (0-1)
func hslToRgb(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}

	var r, g, b float64

	if s == 0 {
		// Grayscale
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		hk := h / 360.0

		r = hueToRgb(p, q, hk+1.0/3.0)
		g = hueToRgb(p, q, hk)
		b = hueToRgb(p, q, hk-1.0/3.0)
	}

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hueToRgb is a helper function for hslToRgb
func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
