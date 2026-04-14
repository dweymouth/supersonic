package browsing

import (
	"image"
	"image/color"
	"log"

	"github.com/boxes-ltd/imaging"
	"github.com/cenkalti/dominantcolor"
	myTheme "github.com/dweymouth/supersonic/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// BackgroundManager handles background blur/gradient for pages
type BackgroundManager struct {
	BackgroundImgA     *canvas.Image
	BackgroundImgB     *canvas.Image
	BackgroundGradient *canvas.LinearGradient

	CachedBlurredImage  image.Image
	CachedDominantColor color.Color

	targetWidth  int
	targetHeight int
}

// NewBackgroundManager creates background widgets
func NewBackgroundManager() *BackgroundManager {
	b := &BackgroundManager{
		BackgroundImgA:     canvas.NewImageFromImage(nil),
		BackgroundImgB:     canvas.NewImageFromImage(nil),
		BackgroundGradient: canvas.NewLinearGradient(color.Transparent, color.Transparent, 0),
	}
	b.BackgroundImgA.Hidden = true
	b.BackgroundImgB.Hidden = true
	// Use Stretch with pre-processed "cover" sized images
	b.BackgroundImgA.FillMode = canvas.ImageFillStretch
	b.BackgroundImgB.FillMode = canvas.ImageFillStretch
	return b
}

// SetTargetSize sets the target resolution for blur processing (call when window size changes)
func (b *BackgroundManager) SetTargetSize(width, height int) {
	if width > 0 && height > 0 {
		b.targetWidth = width
		b.targetHeight = height
	}
}

// ApplyBackground applies background based on mode: "blur", "gradient", or "disabled"
// When mode is empty, it defaults to gradient behavior (for nowplaying page compatibility)
func (b *BackgroundManager) ApplyBackground(img image.Image, mode string, useBlur bool) {
	log.Printf("[BackgroundManager] ApplyBackground: mode=%q, useBlur=%v, img=%v", mode, useBlur, img != nil)
	if img == nil || mode == "disabled" {
		log.Printf("[BackgroundManager] Hiding images (img=%v, mode=%q)", img, mode)
		b.HideImages()
		return
	}

	// Clear cache if image changed
	if img != b.CachedBlurredImage && img != nil {
		b.CachedBlurredImage = nil
		b.CachedDominantColor = nil
	}

	// If blur is requested (explicit mode or useBlur flag), apply blurred background
	if mode == "blur" || useBlur {
		if b.CachedBlurredImage != nil {
			b.applyBlurredBackground(b.CachedBlurredImage)
			return
		}
		go func() {
			// Use detected size or fallback to 1920x1080 if not set
			width, height := b.targetWidth, b.targetHeight
			if width == 0 || height == 0 {
				width, height = 1920, 1080
			}
			resized := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
			blurred := imaging.Blur(resized, 50.0)
			// Flip horizontally as requested
			flipped := imaging.FlipH(blurred)
			b.CachedBlurredImage = flipped
			fyne.Do(func() {
				b.applyBlurredBackground(flipped)
			})
		}()
	} else {
		// Default to gradient (handles mode="gradient" and mode="" for nowplaying)
		if b.CachedDominantColor != nil {
			b.applyGradientBackground(b.CachedDominantColor)
			return
		}
		go func() {
			c := dominantcolor.Find(img)
			b.CachedDominantColor = c
			fyne.Do(func() {
				b.applyGradientBackground(c)
			})
		}()
	}
}

func (b *BackgroundManager) HideImages() {
	if !b.BackgroundImgA.Hidden {
		b.BackgroundImgA.Hide()
		b.BackgroundImgB.Hide()
	}
}

// SetGradientEndColor sets the end color of the gradient (should be theme background)
func (b *BackgroundManager) SetGradientEndColor(c color.Color) {
	b.BackgroundGradient.EndColor = c
	b.BackgroundGradient.Refresh()
}

// ensureGradientEndColor sets the EndColor to theme background if it's currently zero/transparent
func (b *BackgroundManager) ensureGradientEndColor() {
	r, _, _, _ := b.BackgroundGradient.EndColor.RGBA()
	if r == 0 {
		b.BackgroundGradient.EndColor = theme.Color(theme.ColorNameBackground)
	}
}

// ApplyIfChanged checks if image changed, clears cache, and applies background if not nil
func (b *BackgroundManager) ApplyIfChanged(newImg, currentImg image.Image, mode string, useBlur bool) {
	if newImg != currentImg {
		b.CachedBlurredImage = nil
		b.CachedDominantColor = nil
	}
	if newImg != nil {
		b.ApplyBackground(newImg, mode, useBlur)
	}
}

// applyBlurredBackground applies a pre-processed blurred image with smooth transition
func (b *BackgroundManager) applyBlurredBackground(blurred image.Image) {
	log.Printf("[BackgroundManager] applyBlurredBackground called, blurred size=%v", blurred.Bounds())
	// Check if coming from gradient mode (images were hidden)
	fromGradient := b.BackgroundImgA.Hidden && b.BackgroundImgB.Hidden

	// Make gradient transparent so images show through
	b.BackgroundGradient.StartColor = color.Transparent
	b.BackgroundGradient.Refresh()

	if fromGradient {
		// Coming from gradient: just show the image immediately, no crossfade
		b.BackgroundImgA.Hidden = false
		b.BackgroundImgB.Hidden = false
		b.BackgroundImgA.Image = blurred
		b.BackgroundImgB.Image = blurred
		b.BackgroundImgA.Translucency = 0.0
		b.BackgroundImgB.Translucency = 0.0
		b.BackgroundImgA.Refresh()
		b.BackgroundImgB.Refresh()
	} else {
		// Already showing blurred: crossfade to new image
		b.BackgroundImgA.Hidden = false
		b.BackgroundImgB.Hidden = false
		b.BackgroundImgA.Image = b.BackgroundImgB.Image
		b.BackgroundImgB.Image = blurred
		b.BackgroundImgA.Translucency = 0.0
		b.BackgroundImgB.Translucency = 1.0
		b.BackgroundImgA.Refresh()
		b.BackgroundImgB.Refresh()

		fyne.NewAnimation(myTheme.AnimationDurationMedium, func(f float32) {
			b.BackgroundImgA.Translucency = float64(f)
			b.BackgroundImgB.Translucency = float64(1 - f)
			b.BackgroundImgA.Refresh()
			b.BackgroundImgB.Refresh()
		}).Start()
	}
}

// applyGradientBackground applies a gradient with the given dominant color
// The EndColor of the gradient should be set by the caller (page) to match the theme background
func (b *BackgroundManager) applyGradientBackground(c color.Color) {
	b.ensureGradientEndColor()
	if c == b.BackgroundGradient.StartColor && b.BackgroundImgA.Hidden && b.BackgroundImgB.Hidden {
		return
	}

	// Ensure images stay visible during transition
	wasHiddenA := b.BackgroundImgA.Hidden
	wasHiddenB := b.BackgroundImgB.Hidden

	b.BackgroundImgA.Hidden = false
	b.BackgroundImgB.Hidden = false

	// Ensure gradient is visible
	b.BackgroundGradient.Hidden = false
	// Set gradient StartColor to the dominant color (EndColor is maintained by the page/theme)
	b.BackgroundGradient.StartColor = c
	b.BackgroundGradient.Refresh()

	// If images were hidden, make them fully transparent so gradient shows through immediately
	if wasHiddenA {
		b.BackgroundImgA.Translucency = 1.0
	}
	if wasHiddenB {
		b.BackgroundImgB.Translucency = 1.0
	}

	// Animate images fading out to reveal the gradient smoothly
	startTranslucencyA := b.BackgroundImgA.Translucency
	startTranslucencyB := b.BackgroundImgB.Translucency

	fyne.NewAnimation(myTheme.AnimationDurationMedium, func(f float32) {
		// Fade out images to reveal gradient underneath
		if !wasHiddenA {
			b.BackgroundImgA.Translucency = startTranslucencyA + (1-startTranslucencyA)*float64(f)
		}
		if !wasHiddenB {
			b.BackgroundImgB.Translucency = startTranslucencyB + (1-startTranslucencyB)*float64(f)
		}
		b.BackgroundImgA.Refresh()
		b.BackgroundImgB.Refresh()

		// Hide images completely at end
		if f >= 0.99 {
			b.BackgroundImgA.Hide()
			b.BackgroundImgB.Hide()
		}
	}).Start()
}
