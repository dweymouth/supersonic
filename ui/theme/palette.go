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

// GeneratePalette creates a unified color palette from the given configuration
// Uses consistent formulas across all base modes for coherence
// Results are cached with 1-minute TTL for performance
func GeneratePalette(accentHex string, saturation, contrast float64, baseMode string) (*Palette, error) {
	// Check cache first
	cacheKey := generateCacheKey(accentHex, saturation, contrast, baseMode)
	paletteCacheMux.RLock()
	entry, found := paletteCache[cacheKey]
	paletteCacheMux.RUnlock()

	if found && time.Since(entry.timestamp) < paletteCacheTTL {
		return entry.palette, nil
	}

	// Normalize accent color for the mode (ensures visibility)
	normalizer := NewColorNormalizer()
	normalizedHex := normalizer.NormalizeForMode(accentHex, baseMode)
	accent, _ := hexToColor(normalizedHex)
	accentH, accentS, accentL := rgbToHslColor(accent)

	// Apply saturation/contrast sliders to accent
	if saturation != 1.0 {
		accentS = clampFloat(accentS*saturation, 0, 1)
	}
	if contrast != 1.0 {
		// Contrast affects lightness: >1 = more extreme, <1 = closer to mid
		midpoint := 0.5
		accentL = midpoint + (accentL-midpoint)*contrast
		accentL = clampFloat(accentL, 0.15, 0.85)
	}
	accent = hslToRgb(accentH, accentS, accentL)

	// Determine mode characteristics
	isLight := strings.ToLower(baseMode) == "light"
	isBlack := strings.ToLower(baseMode) == "black"

	// Unified palette generation using consistent ratios
	// All values derived from accent hue with mode-specific lightness
	var bgL, surfL, pageBgL, pageHeaderL, listHeaderL, textPrimaryL, textSecondaryL float64
	var bgSat float64

	if isLight {
		// Light mode: light backgrounds, darker text for contrast
		bgL = 0.94
		surfL = 1.0 // Pure white for surfaces
		pageBgL = 0.96
		pageHeaderL = 0.90
		listHeaderL = 0.92
		textPrimaryL = 0.08   // Darker for better contrast (was 0.12)
		textSecondaryL = 0.28 // Darker gray (was 0.40)
		bgSat = 0.03
	} else if isBlack {
		// Black/AMOLED mode: pure black to dark grey
		bgL = 0.02
		surfL = 0.10
		pageBgL = 0.04
		pageHeaderL = 0.14
		listHeaderL = 0.12
		textPrimaryL = 0.95
		textSecondaryL = 0.65
		bgSat = 0.04
	} else {
		// Dark mode (default): dark greys
		bgL = 0.10
		surfL = 0.16
		pageBgL = 0.08
		pageHeaderL = 0.20
		listHeaderL = 0.18
		textPrimaryL = 0.94
		textSecondaryL = 0.62
		bgSat = 0.06
	}

	// Apply contrast adjustment to background lightness
	// Higher contrast = more separation from middle grey
	contrastDelta := (contrast - 1.0) * 0.08
	if isLight {
		// Light mode: higher contrast = lighter backgrounds
		bgL = clampFloat(bgL+contrastDelta, 0.85, 1.0)
		surfL = clampFloat(surfL+contrastDelta*0.5, 0.90, 1.0)
	} else {
		// Dark mode: higher contrast = darker backgrounds
		bgL = clampFloat(bgL-contrastDelta, 0.0, 0.25)
		surfL = clampFloat(surfL-contrastDelta*0.5, 0.05, 0.30)
	}

	// Generate colors from accent hue with calculated lightness
	bg := hslToRgb(accentH, bgSat, bgL)
	surf := hslToRgb(accentH, bgSat, surfL)
	pageBg := hslToRgb(accentH, bgSat, pageBgL)
	pageHeader := hslToRgb(accentH, bgSat, pageHeaderL)
	listHeader := hslToRgb(accentH, bgSat*0.8, listHeaderL)

	// Text colors: accent hue with low saturation for subtle tint
	textSat := 0.08
	if isLight {
		textSat = 0.12 // Slightly more saturation in light mode for visibility
	}
	textPrimaryRGB := hslToRgb(accentH, textSat, textPrimaryL)
	textSecondaryRGB := hslToRgb(accentH, textSat*0.7, textSecondaryL)

	// Ensure WCAG contrast compliance
	textPrimary, _ := EnsureContrast(textPrimaryRGB, surf, 4.5)
	textSecondary, _ := EnsureContrast(textSecondaryRGB, surf, 3.0) // AA for large text

	// Calculate text on accent
	textOnAccent := calculateTextOnAccent(accent)

	// Surface hover: blend accent with surface
	hoverRatio := 0.22
	if !isLight {
		hoverRatio = 0.28 // Stronger hover in dark modes
	}
	surfaceHover := blendColors(accent, surf, hoverRatio)

	// Menu background: subtler blend
	menuBackground := blendColors(accent, surf, hoverRatio*0.5)

	// Success/Danger: keep standard but tint slightly with accent hue
	success := hslToRgb(140.0, 0.60, 0.45) // Green
	danger := hslToRgb(0.0, 0.70, 0.50)    // Red

	// For hyperlinks in light mode, darken the accent to ensure contrast against light backgrounds
	hyperlink := accent
	if isLight {
		// Darken accent for hyperlinks in light mode (70% of original luminance)
		_, accentS, accentL := rgbToHslColor(accent)
		hyperlink = hslToRgb(accentH, accentS, accentL*0.65)
		// Ensure minimum darkness for hyperlinks
		hLinkLum := (0.299*float64(hyperlink.R) + 0.587*float64(hyperlink.G) + 0.114*float64(hyperlink.B)) / 255.0
		if hLinkLum > 0.35 {
			hyperlink = hslToRgb(accentH, accentS, 0.35)
		}
	}

	palette := &Palette{
		Accent:         accent,
		Background:     bg,
		Surface:        surf,
		SurfaceHover:   surfaceHover,
		MenuBackground: menuBackground,
		TextPrimary:    textPrimary,
		TextSecondary:  textSecondary,
		TextOnAccent:   textOnAccent,
		Success:        success,
		Danger:         danger,
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
func ensureContrast(textColor color.RGBA, bgLum float64) color.RGBA {
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
