package widgets

import (
	"image"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
)

type WaveformSeekbar struct {
	widget.BaseWidget

	imgColorL   color.Color
	imgColorR   color.Color
	imgProgress float64

	img *canvas.Image
}

func NewWaveformSeekbar() *WaveformSeekbar {
	w := &WaveformSeekbar{
		img: canvas.NewImageFromImage(nil),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *WaveformSeekbar) UpdateImage(img *backend.WaveformImage) {
	w.img.Image = img
	w.Refresh()
}

func (w *WaveformSeekbar) Refresh() {
	prm, fg := w.getThemeColors()
	w.recolorImage(prm, fg, w.imgProgress)
	w.img.Refresh()
	//w.BaseWidget.Refresh()
}

func (w *WaveformSeekbar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.New(layout.NewCustomPaddedLayout(4, 4, 0, 0), w.img),
	)
}

// SetProgress sets how much of the seekbar has been played
// (ratio from 0 to 1)
func (w *WaveformSeekbar) SetProgress(v float64) {
	prm, fg := w.getThemeColors()
	w.recolorImage(prm, fg, v)
	w.Refresh()
}

func (w *WaveformSeekbar) getThemeColors() (primary, foreground color.Color) {
	th := w.Theme()
	vnt := fyne.CurrentApp().Settings().ThemeVariant()
	primary = th.Color(theme.ColorNamePrimary, vnt)
	foreground = th.Color(theme.ColorNameForeground, vnt)
	return primary, foreground
}

func (w *WaveformSeekbar) recolorImage(cL, cR color.Color, progress float64) {
	if w.img.Image == nil {
		return
	}
	if w.imgColorL == cL && w.imgColorR == cR && w.imgProgress == progress {
		return
	}

	_r, _g, _b, _ := cL.RGBA()
	rL, gL, bL := byte(_r>>8), byte(_g>>8), byte(_b>>8)
	_r, _g, _b, _ = cR.RGBA()
	rR, gR, bR := byte(_r>>8), byte(_g>>8), byte(_b>>8)

	// TODO- smartly figure out which pixels we need
	// to update for different scenarios (e.g. progress update only)
	// and not iterate the whole thing every time
	img := w.img.Image.(*image.NRGBA)
	bnds := img.Rect.Bounds()
	thresholdPixel := int(float64(bnds.Dx()) * progress)
	log.Println("progress = ", progress, " bounds Dx = ", bnds.Dx())
	log.Println("thresholdPixel = ", thresholdPixel)
	for x := 0; x < bnds.Dx(); x++ {
		for y := 0; y < bnds.Dy(); y++ {
			if x < thresholdPixel {
				setPixelRGB(img, x, y, rL, gL, bL)
			} else {
				setPixelRGB(img, x, y, rR, gR, bR)
			}
		}
	}

	w.imgColorL, w.imgColorR = cL, cR
	w.imgProgress = progress
}

func setPixelRGB(img *image.NRGBA, x, y int, r, g, b byte) {
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = r
	img.Pix[offset+1] = g
	img.Pix[offset+2] = b
}
