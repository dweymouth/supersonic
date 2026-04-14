package theme

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
)

// ColorExtractor extracts vibrant accent colors from images
type ColorExtractor struct {
	// SampleRate: process 1 pixel every N pixels (default 10)
	SampleRate int
}

// NewColorExtractor creates a new extractor with default settings
func NewColorExtractor() *ColorExtractor {
	return &ColorExtractor{
		SampleRate: 10, // Sample every 10th pixel for performance
	}
}

// ExtractAccentFromImage extracts a vibrant accent color from an image.
// Returns hex color string (e.g., "#FF6A00") or error.
func (ce *ColorExtractor) ExtractAccentFromImage(img image.Image) (string, error) {
	if img == nil {
		return "", fmt.Errorf("nil image")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return "", fmt.Errorf("empty image")
	}

	// Collect color samples
	type colorScore struct {
		r, g, b    uint8
		h, s, l    float64
		saturation float64
		vibrancy   float64
	}

	var samples []colorScore

	// Sample pixels with stride
	stride := ce.SampleRate
	if stride < 1 {
		stride = 1
	}

	for y := 0; y < height; y += stride {
		for x := 0; x < width; x += stride {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Convert from 16-bit to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			h, s, l := rgbToHslFloat(r8, g8, b8)

			// Calculate vibrancy: prefer saturated colors with medium luminance
			// Penalize grays (low saturation), blacks (low luminance), and whites (high luminance)
			vibrancy := s * (1.0 - math.Abs(l-0.5)*2.0) // Max at l=0.5, zero at l=0 or l=1

			samples = append(samples, colorScore{
				r: r8, g: g8, b: b8,
				h: h, s: s, l: l,
				saturation: s,
				vibrancy:   vibrancy,
			})
		}
	}

	if len(samples) == 0 {
		return "", fmt.Errorf("no samples collected")
	}

	// Check if image is essentially monochrome (B&W or desaturated)
	// Calculate average saturation and luminance across all samples
	var totalSaturation, totalLuminance float64
	for _, s := range samples {
		totalSaturation += s.saturation
		totalLuminance += s.l
	}
	avgSaturation := totalSaturation / float64(len(samples))
	avgLuminance := totalLuminance / float64(len(samples))

	// If image is essentially black & white (low saturation), extract subtle metallic accent
	// The accent is desaturated but has enough lightness to be visible in dark modes
	if avgSaturation < 0.25 {
		// For B&W images, create a subtle warm/cool metallic accent
		// Light B&W -> warm silver (amber tint)
		// Dark B&W -> steel blue (cool tint)
		var hue float64
		if avgLuminance > 0.5 {
			hue = 40.0 // Warm amber for light images
		} else {
			hue = 210.0 // Cool blue for dark images
		}
		saturation := 0.001 // Very subtle tint
		// Always use light luminance so it's visible in dark/black modes
		// Force light grays (0.70-0.85) regardless of image luminance
		lightness := 0.70 + (avgLuminance * 0.15) // Range 0.70-0.85
		accent := hslToRgb(hue, saturation, lightness)
		return fmt.Sprintf("#%02X%02X%02X", accent.R, accent.G, accent.B), nil
	}

	// Filter out low-quality colors (grays, near-blacks, near-whites)
	var candidates []colorScore
	for _, s := range samples {
		// Require minimum saturation and reasonable luminance
		if s.saturation > 0.25 && s.l > 0.15 && s.l < 0.85 {
			candidates = append(candidates, s)
		}
	}

	// If no good candidates, relax constraints
	if len(candidates) == 0 {
		for _, s := range samples {
			if s.saturation > 0.15 && s.l > 0.1 && s.l < 0.9 {
				candidates = append(candidates, s)
			}
		}
	}

	// Still no candidates? Use any non-extreme color
	if len(candidates) == 0 {
		for _, s := range samples {
			if s.l > 0.05 && s.l < 0.95 {
				candidates = append(candidates, s)
			}
		}
	}

	if len(candidates) == 0 {
		// Fallback to average of all samples
		var sumR, sumG, sumB int64
		for _, s := range samples {
			sumR += int64(s.r)
			sumG += int64(s.g)
			sumB += int64(s.b)
		}
		avgR := uint8(sumR / int64(len(samples)))
		avgG := uint8(sumG / int64(len(samples)))
		avgB := uint8(sumB / int64(len(samples)))
		return fmt.Sprintf("#%02X%02X%02X", avgR, avgG, avgB), nil
	}

	// Group colors by hue buckets for finding dominant colors
	const hueBuckets = 12 // 30 degrees each
	buckets := make([][]colorScore, hueBuckets)

	for _, c := range candidates {
		bucket := int(c.h/30.0) % hueBuckets
		buckets[bucket] = append(buckets[bucket], c)
	}

	// Score each bucket by total vibrancy
	type bucketScore struct {
		index    int
		vibrancy float64
		count    int
	}

	var scores []bucketScore
	for i, bucket := range buckets {
		if len(bucket) > 0 {
			var totalVibrancy float64
			for _, c := range bucket {
				totalVibrancy += c.vibrancy
			}
			scores = append(scores, bucketScore{
				index:    i,
				vibrancy: totalVibrancy,
				count:    len(bucket),
			})
		}
	}

	// Sort by vibrancy descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].vibrancy > scores[j].vibrancy
	})

	// Get the best bucket and average its colors
	if len(scores) == 0 {
		return "", fmt.Errorf("no color buckets found")
	}

	bestBucket := buckets[scores[0].index]

	// Average colors in the best bucket, weighted by vibrancy
	var sumR, sumG, sumB, weight float64
	for _, c := range bestBucket {
		w := c.vibrancy + 0.1 // Base weight to avoid zero
		sumR += float64(c.r) * w
		sumG += float64(c.g) * w
		sumB += float64(c.b) * w
		weight += w
	}

	if weight == 0 {
		return "", fmt.Errorf("zero weight")
	}

	avgR := uint8(sumR / weight)
	avgG := uint8(sumG / weight)
	avgB := uint8(sumB / weight)

	// Boost saturation slightly for more vibrant accent
	avgR, avgG, avgB = boostSaturation(avgR, avgG, avgB, 1.15)

	return fmt.Sprintf("#%02X%02X%02X", avgR, avgG, avgB), nil
}

// rgbToHslFloat converts RGB to HSL with float64 precision
func rgbToHslFloat(r, g, b uint8) (h, s, l float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	// Lightness
	l = (max + min) / 2.0

	// Saturation
	if delta == 0 {
		s = 0
	} else {
		s = delta / (1.0 - math.Abs(2.0*l-1.0))
	}

	// Hue
	if delta == 0 {
		h = 0
	} else {
		switch max {
		case rf:
			h = math.Mod((gf-bf)/delta, 6.0)
			if gf < bf {
				h += 6.0
			}
		case gf:
			h = (bf-rf)/delta + 2.0
		case bf:
			h = (rf-gf)/delta + 4.0
		}
		h *= 60.0
	}

	return h, s, l
}

// boostSaturation increases the saturation of a color
func boostSaturation(r, g, b uint8, factor float64) (uint8, uint8, uint8) {
	h, s, l := rgbToHslFloat(r, g, b)

	// Boost saturation
	s = math.Min(s*factor, 1.0)

	// Convert back to RGB
	c := (1.0 - math.Abs(2.0*l-1.0)) * s
	x := c * (1.0 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := l - c/2.0

	var rf, gf, bf float64

	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	return uint8((rf + m) * 255.0), uint8((gf + m) * 255.0), uint8((bf + m) * 255.0)
}

// interpolateColor linearly interpolates between two RGB colors
func interpolateColor(c1, c2 color.Color, t float64) color.Color {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	// Convert from 16-bit to 8-bit and interpolate
	r := uint8((float64(r1>>8)*(1-t) + float64(r2>>8)*t))
	g := uint8((float64(g1>>8)*(1-t) + float64(g2>>8)*t))
	b := uint8((float64(b1>>8)*(1-t) + float64(b2>>8)*t))
	a := uint8((float64(a1>>8)*(1-t) + float64(a2>>8)*t))

	return color.RGBA{R: r, G: g, B: b, A: a}
}
