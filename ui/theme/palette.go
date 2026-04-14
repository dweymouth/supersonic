package theme

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"
	"time"
)

// Cache entry with timestamp for TTL
type paletteCacheEntry struct {
	palette   *Palette
	timestamp time.Time
}

// Global cache with 1-minute TTL
var (
	paletteCache    = make(map[string]paletteCacheEntry)
	paletteCacheMux sync.RWMutex
	paletteCacheTTL = 1 * time.Minute
)

// generateCacheKey creates a unique key for the palette parameters
func generateCacheKey(accentHex string, saturation, contrast float64, baseMode string) string {
	return fmt.Sprintf("%s|%.3f|%.3f|%s", accentHex, saturation, contrast, baseMode)
}

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
		return SliderRanges{SatMin: 0.0, SatMax: 0.5, ContrastMin: 0.25, ContrastMax: 1.0}
	case "black":
		return SliderRanges{SatMin: 0.0, SatMax: 0.75, ContrastMin: 0.25, ContrastMax: 1.0}
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
// Results are cached with 1-minute TTL for performance
func GeneratePalette(accentHex string, saturation, contrast float64, baseMode string) (*Palette, error) {
	// Check cache first
	cacheKey := generateCacheKey(accentHex, saturation, contrast, baseMode)
	paletteCacheMux.RLock()
	entry, found := paletteCache[cacheKey]
	paletteCacheMux.RUnlock()

	if found && time.Since(entry.timestamp) < paletteCacheTTL {
		// Cache hit and not expired
		return entry.palette, nil
	}

	accent, err := hexToColor(accentHex)
	if err != nil {
		return nil, fmt.Errorf("invalid accent color: %w", err)
	}

	// Convert accent to HSL for dynamic adjustments
	accentH, accentS, accentL := rgbToHsl(accent)

	// Adapt accent color for dark modes (black/grey) to ensure visibility
	// When extracting from cover art, dark/unsaturated colors get lost on dark backgrounds
	baseModeLower := strings.ToLower(baseMode)
	if baseModeLower == "black" || baseModeLower == "grey" {
		// For dark modes: ensure accent has minimum brightness
		// Dark muddy colors (< 0.35 lightness) need to be lightened
		if accentL < 0.40 {
			// Lighten dark colors significantly for visibility on dark backgrounds
			accentL = 0.35 + (accentL * 0.4) // Map 0.0-0.4 -> 0.35-0.51
		}
		if accentL < 0.50 {
			// Further boost for very dark extracted colors
			accentL = 0.45 + ((accentL - 0.35) * 0.5)
		}
		// For light colors (> 0.60 lightness), keep them as-is
		// Bright accents like yellow need to stay bright for visibility
		// No clamping applied - user gets the color they selected
		// Boost saturation for dark modes - but NOT for B&W/monochrome colors
		// If saturation is very low (< 0.05), the color is intentionally desaturated (B&W)
		// Only boost colors that already have some saturation
		if accentS >= 0.05 && accentS < 0.50 {
			accentS = 0.50 + (accentS * 0.3)
		} else if accentS >= 0.05 && accentS < 0.70 {
			accentS = 0.60 + ((accentS - 0.50) * 0.5)
		}
		// Clamp and apply
		accentS = clampFloat(accentS, 0, 1)
		accentL = clampFloat(accentL, 0, 1)
		accent = hslToRgb(accentH, accentS, accentL)
		// Recalculate HSL after modification
		accentH, accentS, accentL = rgbToHsl(accent)
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
		// Hybrid: black (AMOLED) at high contrast, grey at low contrast
		// Interpolate based on contrast: 0.0-0.5 -> grey-like, 0.5-1.0 -> black
		greyFactor := clampFloat((contrast-0.5)*2, 0, 1) // 0.0 at low contrast, 1.0 at high contrast

		// Black values (high contrast)
		bgLBlack, surfLBlack, pageBgLBlack := 0.04, 0.10, 0.03
		pageHeaderLBlack, listHeaderLBlack := 0.14, 0.12
		satPrimaryBlack, satSecondaryBlack := 0.06, 0.05
		lumPrimaryBlack, lumSecondaryBlack := 0.98, 0.70

		// Grey values (low contrast)
		bgLGrey, surfLGrey, pageBgLGrey := 0.12, 0.18, 0.10
		pageHeaderLGrey, listHeaderLGrey := 0.22, 0.20
		satPrimaryGrey, satSecondaryGrey := 0.08, 0.06
		lumPrimaryGrey, lumSecondaryGrey := 0.96, 0.65

		// Interpolate
		bgL = bgLGrey + (bgLBlack-bgLGrey)*greyFactor
		surfL = surfLGrey + (surfLBlack-surfLGrey)*greyFactor
		pageBgL = pageBgLGrey + (pageBgLBlack-pageBgLGrey)*greyFactor
		pageHeaderL = pageHeaderLGrey + (pageHeaderLBlack-pageHeaderLGrey)*greyFactor
		listHeaderL = listHeaderLGrey + (listHeaderLBlack-listHeaderLGrey)*greyFactor

		satPrimary := satPrimaryGrey + (satPrimaryBlack-satPrimaryGrey)*greyFactor
		satSecondary := satSecondaryGrey + (satSecondaryBlack-satSecondaryGrey)*greyFactor
		lumPrimary := lumPrimaryGrey + (lumPrimaryBlack-lumPrimaryGrey)*greyFactor
		lumSecondary := lumSecondaryGrey + (lumSecondaryBlack-lumSecondaryGrey)*greyFactor

		textPrimaryRGB = hslToRgb(accentH, satPrimary, lumPrimary)
		textSecondaryRGB = hslToRgb(accentH, satSecondary, lumSecondary)
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

	// Ensure text has enough contrast against Surface background
	// Calculate surface luminance and adjust text colors if needed
	surfLum := (float64(surf.R)*0.299 + float64(surf.G)*0.587 + float64(surf.B)*0.114) / 255.0
	textPrimary := ensureContrast(textPrimaryRGB, surf, surfLum)
	textSecondary := ensureContrast(textSecondaryRGB, surf, surfLum)

	// Apply saturation to accent (slider affects accent intensity)
	if saturation != 1.0 {
		accent = applySaturationToHSL(accentH, accentS, accentL, saturation)
		accentH, accentS, accentL = rgbToHsl(accent)
	}

	// Apply contrast adjustment to accent
	accent = applyContrast(accent, contrast, baseMode == "light")

	// Ensure minimum contrast between Surface and Accent for hyperlink readability
	accent = ensureSurfaceContrast(accent, surf, isLight)

	// Calculate text color on accent based on luminance
	textOnAccent := calculateTextOnAccent(accent)

	// Generate surface hover color by blending accent with surface
	// Blend ratio is inversely proportional to surface lightness:
	// - Light surfaces (light mode): strong hover (24-28% accent)
	// - Dark surfaces (dark modes): stronger hover (32-36% accent)
	hoverBlendRatio := 0.36 - (surfLum * 0.08) // 0.36 when black, 0.28 when white
	surfaceHover := blendColors(accent, surf, clampFloat(hoverBlendRatio, 0.24, 0.40))

	// Menu background: subtle accent tint, half the intensity of SurfaceHover
	// - Light surfaces: ~12-14% accent
	// - Dark surfaces: ~16-18% accent
	menuBlendRatio := 0.18 - (surfLum * 0.04) // 0.18 when black, 0.14 when white
	menuBackground := blendColors(accent, surf, clampFloat(menuBlendRatio, 0.12, 0.20))

	// Set hyperlink color to accent
	hyperlink := accent

	palette := &Palette{
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
	}

	// Store in cache
	paletteCacheMux.Lock()
	paletteCache[cacheKey] = paletteCacheEntry{
		palette:   palette,
		timestamp: time.Now(),
	}
	paletteCacheMux.Unlock()

	return palette, nil
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
// For bright colors, preserves original to avoid crushing bright accents
func applyContrast(c color.RGBA, contrast float64, isLightMode bool) color.RGBA {
	// Calculate luminance (0-255 scale)
	luminance := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)

	// For bright colors (> 55% luminance), skip contrast adjustment entirely
	// This preserves bright accents like cyan, yellow, and green (#00FF00 luminance=150)
	if luminance > 140 { // 140 = ~0.55 * 255
		return c // Return original color unchanged
	}

	var multiplier float64
	if isLightMode {
		// Light mode: contrast slider 0.0-0.5 maps to multiplier 0.92-1.0
		// This provides very subtle adjustment, preserving color brightness
		multiplier = 0.92 + (contrast * 0.16) // 0.0->0.92, 0.5->1.0
	} else {
		// Dark mode: contrast slider 0.0-1.0 maps to multiplier 0.5-1.0
		// Minimum 0.5 ensures color is never crushed to black
		multiplier = 0.5 + (contrast * 0.5) // 0.0->0.5, 1.0->1.0
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

// ensureContrast adjusts text color to ensure minimum contrast against background
// Returns darkened text for light backgrounds, lightened text for dark backgrounds
func ensureContrast(textColor, bgColor color.RGBA, bgLum float64) color.RGBA {
	// Calculate text luminance
	textLum := (float64(textColor.R)*0.299 + float64(textColor.G)*0.587 + float64(textColor.B)*0.114) / 255.0

	// Determine if we need dark or light text based on background
	if bgLum > 0.5 {
		// Light background: ensure text is dark enough (luminance < 0.3)
		if textLum > 0.3 {
			// Text too light for light background, darken it
			factor := 0.3 / textLum
			return color.RGBA{
				R: uint8(float64(textColor.R) * factor),
				G: uint8(float64(textColor.G) * factor),
				B: uint8(float64(textColor.B) * factor),
				A: textColor.A,
			}
		}
	} else {
		// Dark background: ensure text is light enough (luminance > 0.6)
		if textLum < 0.6 {
			// Text too dark for dark background, lighten it
			factor := 0.6 / textLum
			if factor > 3.0 {
				factor = 3.0 // Cap maximum brightening
			}
			return color.RGBA{
				R: uint8(clampFloat(float64(textColor.R)*factor, 0, 255)),
				G: uint8(clampFloat(float64(textColor.G)*factor, 0, 255)),
				B: uint8(clampFloat(float64(textColor.B)*factor, 0, 255)),
				A: textColor.A,
			}
		}
	}
	return textColor
}

// ensureSurfaceContrast guarantees minimum contrast between Surface and Accent colors
// This ensures hyperlinks remain readable regardless of the selected accent color
func ensureSurfaceContrast(accent, surface color.Color, isLight bool) color.RGBA {
	ar, ag, ab, _ := accent.RGBA()
	sr, sg, sb, _ := surface.RGBA()

	// Calculate luminances (0-1 range)
	accentLum := (0.299*float64(ar) + 0.587*float64(ag) + 0.114*float64(ab)) / 65535.0
	surfaceLum := (0.299*float64(sr) + 0.587*float64(sg) + 0.114*float64(sb)) / 65535.0

	// Minimum contrast threshold (30% luminance difference)
	const minContrast = 0.30

	diff := accentLum - surfaceLum
	if diff < 0 {
		diff = -diff
	}

	// If contrast is sufficient, return accent unchanged
	if diff >= minContrast {
		return color.RGBA{R: uint8(ar >> 8), G: uint8(ag >> 8), B: uint8(ab >> 8), A: 255}
	}

	// Need to adjust accent for more contrast
	accentRGB := color.RGBA{R: uint8(ar >> 8), G: uint8(ag >> 8), B: uint8(ab >> 8), A: 255}

	if isLight {
		// Light mode: Surface is light (~0.88-0.96), accent needs to be darker
		// Darken accent significantly to contrast against light surface
		return darkenColor(accentRGB, 0.6).(color.RGBA)
	}
	// Dark mode: Surface is dark (~0.10-0.18), accent needs to be lighter
	// Lighten accent significantly to contrast against dark surface
	return brightenColor(accentRGB, 0.8).(color.RGBA)
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
