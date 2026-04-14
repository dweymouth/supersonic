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
}

// NewBackgroundManager creates background widgets
func NewBackgroundManager() *BackgroundManager {
	log.Printf("[BackgroundManager] Creating new instance")
	b := &BackgroundManager{
		BackgroundImgA:     canvas.NewImageFromImage(nil),
		BackgroundImgB:     canvas.NewImageFromImage(nil),
		BackgroundGradient: canvas.NewLinearGradient(color.Transparent, color.Transparent, 0),
	}
	b.BackgroundImgA.Hidden = true
	b.BackgroundImgB.Hidden = true
	// CRITICAL: Set FillMode to Stretch so images cover the entire container
	b.BackgroundImgA.FillMode = canvas.ImageFillStretch
	b.BackgroundImgB.FillMode = canvas.ImageFillStretch
	log.Printf("[BackgroundManager] FillMode set to Stretch for both images")
	return b
}

// ApplyBackground applies background based on mode: "blur", "gradient", or "disabled"
// When mode is empty, it defaults to gradient behavior (for nowplaying page compatibility)
func (b *BackgroundManager) ApplyBackground(img image.Image, mode string, useBlur bool) {
	log.Printf("[BackgroundManager] ApplyBackground called: mode=%q, useBlur=%v, img=%v", mode, useBlur, img != nil)
	if img == nil || mode == "disabled" {
		log.Printf("[BackgroundManager] Hiding images (img=%v, mode=%q)", img, mode)
		b.hideImages()
		return
	}

	// Clear cache if image changed
	if img != b.CachedBlurredImage && img != nil {
		b.CachedBlurredImage = nil
		b.CachedDominantColor = nil
	}

	// If blur is requested (explicit mode or useBlur flag), apply blurred background
	if mode == "blur" || useBlur {
		log.Printf("[BackgroundManager] Applying blur mode")
		if b.CachedBlurredImage != nil {
			b.applyBlurredBackground(b.CachedBlurredImage)
			return
		}
		go func() {
			resized := imaging.Resize(img, 300, 0, imaging.NearestNeighbor)
			blurred := imaging.Blur(resized, 10.0)
			b.CachedBlurredImage = blurred
			fyne.Do(func() {
				b.applyBlurredBackground(blurred)
			})
		}()
	} else {
		// Default to gradient (handles mode="gradient" and mode="" for nowplaying)
		log.Printf("[BackgroundManager] Applying gradient mode")
		if b.CachedDominantColor != nil {
			log.Printf("[BackgroundManager] Using cached dominant color")
			b.applyGradientBackground(b.CachedDominantColor)
			return
		}
		go func() {
			log.Printf("[BackgroundManager] Finding dominant color...")
			c := dominantcolor.Find(img)
			log.Printf("[BackgroundManager] Dominant color found: %v", c)
			b.CachedDominantColor = c
			fyne.Do(func() {
				b.applyGradientBackground(c)
			})
		}()
	}
}

func (b *BackgroundManager) hideImages() {
	log.Printf("[BackgroundManager] hideImages called")
	if !b.BackgroundImgA.Hidden {
		log.Printf("[BackgroundManager] Hiding BackgroundImgA")
		b.BackgroundImgA.Hide()
		b.BackgroundImgB.Hide()
	} else {
		log.Printf("[BackgroundManager] Images already hidden")
	}
}

// SetGradientEndColor sets the end color of the gradient (should be theme background)
func (b *BackgroundManager) SetGradientEndColor(c color.Color) {
	r, g, bl, a := c.RGBA()
	log.Printf("[BackgroundManager] SetGradientEndColor: RGBA(%d, %d, %d, %d)", r>>8, g>>8, bl>>8, a>>8)
	b.BackgroundGradient.EndColor = c
	b.BackgroundGradient.Refresh()
}

// ensureGradientEndColor sets the EndColor to theme background if it's currently zero/transparent
func (b *BackgroundManager) ensureGradientEndColor() {
	r, _, _, _ := b.BackgroundGradient.EndColor.RGBA()
	if r == 0 {
		c := theme.Color(theme.ColorNameBackground)
		log.Printf("[BackgroundManager] Auto-setting EndColor to theme background: %v", c)
		b.BackgroundGradient.EndColor = c
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
	log.Printf("[BackgroundManager] applyBlurredBackground called, blurred=%v", blurred != nil)
	log.Printf("[BackgroundManager] Image sizes: imgA=%v, imgB=%v, gradient=%v",
		b.BackgroundImgA.Size(), b.BackgroundImgB.Size(), b.BackgroundGradient.Size())
	// Check if coming from gradient mode (images were hidden)
	fromGradient := b.BackgroundImgA.Hidden && b.BackgroundImgB.Hidden
	log.Printf("[BackgroundManager] fromGradient=%v, imgA.Hidden=%v, imgB.Hidden=%v", fromGradient, b.BackgroundImgA.Hidden, b.BackgroundImgB.Hidden)

	// Make gradient transparent so images show through
	log.Printf("[BackgroundManager] Setting StartColor to Transparent")
	b.BackgroundGradient.StartColor = color.Transparent
	b.BackgroundGradient.Refresh()

	if fromGradient {
		// Coming from gradient: just show the image immediately, no crossfade
		log.Printf("[BackgroundManager] Showing blurred image immediately (from gradient)")
		log.Printf("[BackgroundManager] Setting BackgroundImgA and BackgroundImgB to visible")
		b.BackgroundImgA.Hidden = false
		b.BackgroundImgB.Hidden = false
		log.Printf("[BackgroundManager] Setting BackgroundImgA and BackgroundImgB images to blurred")
		b.BackgroundImgA.Image = blurred
		b.BackgroundImgB.Image = blurred
		log.Printf("[BackgroundManager] Setting BackgroundImgA and BackgroundImgB translucency to 0.0")
		b.BackgroundImgA.Translucency = 0.0
		b.BackgroundImgB.Translucency = 0.0
		log.Printf("[BackgroundManager] Refreshing BackgroundImgA and BackgroundImgB")
		b.BackgroundImgA.Refresh()
		b.BackgroundImgB.Refresh()
	} else {
		// Already showing blurred: crossfade to new image
		log.Printf("[BackgroundManager] Crossfading to new blurred image")
		b.BackgroundImgA.Hidden = false
		b.BackgroundImgB.Hidden = false
		b.BackgroundImgA.Image = b.BackgroundImgB.Image
		b.BackgroundImgB.Image = blurred
		b.BackgroundImgA.Translucency = 0.0
		b.BackgroundImgB.Translucency = 1.0
		b.BackgroundImgA.Refresh()
		b.BackgroundImgB.Refresh()

		log.Printf("[BackgroundManager] Starting crossfade animation")
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
	log.Printf("[BackgroundManager] applyGradientBackground sizes: imgA=%v, imgB=%v, gradient=%v",
		b.BackgroundImgA.Size(), b.BackgroundImgB.Size(), b.BackgroundGradient.Size())
	r, g, bl, a := c.RGBA()
	log.Printf("[BackgroundManager] applyGradientBackground: dominant color RGBA(%d, %d, %d, %d)", r>>8, g>>8, bl>>8, a>>8)
	log.Printf("[BackgroundManager] Current gradient: StartColor=%v, EndColor=%v", b.BackgroundGradient.StartColor, b.BackgroundGradient.EndColor)
	if c == b.BackgroundGradient.StartColor && b.BackgroundImgA.Hidden && b.BackgroundImgB.Hidden {
		log.Printf("[BackgroundManager] Skipping gradient update (same color, images hidden)")
		return
	}

	// Ensure images stay visible during transition
	wasHiddenA := b.BackgroundImgA.Hidden
	wasHiddenB := b.BackgroundImgB.Hidden
	log.Printf("[BackgroundManager] wasHiddenA=%v, wasHiddenB=%v", wasHiddenA, wasHiddenB)

	b.BackgroundImgA.Hidden = false
	b.BackgroundImgB.Hidden = false
	log.Printf("[BackgroundManager] Set images to Hidden=false")

	// Ensure gradient is visible
	b.BackgroundGradient.Hidden = false
	// Set gradient StartColor to the dominant color (EndColor is maintained by the page/theme)
	log.Printf("[BackgroundManager] Setting StartColor to dominant color")
	b.BackgroundGradient.StartColor = c
	b.BackgroundGradient.Refresh()

	// If images were hidden, make them fully transparent so gradient shows through immediately
	if wasHiddenA {
		log.Printf("[BackgroundManager] Setting imgA Translucency to 1.0 (was hidden)")
		b.BackgroundImgA.Translucency = 1.0
	}
	if wasHiddenB {
		log.Printf("[BackgroundManager] Setting imgB Translucency to 1.0 (was hidden)")
		b.BackgroundImgB.Translucency = 1.0
	}

	// Animate images fading out to reveal the gradient smoothly
	startTranslucencyA := b.BackgroundImgA.Translucency
	startTranslucencyB := b.BackgroundImgB.Translucency
	log.Printf("[BackgroundManager] Starting animation with translucency A=%.2f, B=%.2f", startTranslucencyA, startTranslucencyB)

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
			log.Printf("[BackgroundManager] Animation complete, hiding images")
			b.BackgroundImgA.Hide()
			b.BackgroundImgB.Hide()
		}
	}).Start()
}
