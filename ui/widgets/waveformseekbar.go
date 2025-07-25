package widgets

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

type WaveformSeekbar struct {
	widget.BaseWidget

	OnSeeked func(float64)

	imgColorL        color.Color
	imgColorR        color.Color
	imgProgressPixel int

	img    *canvas.Image
	cursor *myTheme.ThemedRectangle
}

func NewWaveformSeekbar() *WaveformSeekbar {
	w := &WaveformSeekbar{
		img: &canvas.Image{
			ScaleMode: canvas.ImageScaleFastest,
		},
		cursor: myTheme.NewThemedRectangle(theme.ColorNameForeground),
	}
	w.cursor.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (w *WaveformSeekbar) UpdateImage(img *backend.WaveformImage) {
	w.img.Image = img
	prm, fg := w.getThemeColors()
	w.recolorImage(prm, fg, w.imgProgressPixel)
	w.Refresh()
}

func (w *WaveformSeekbar) Refresh() {
	w.cursor.Resize(fyne.NewSize(1.5, w.Size().Height-4))
	w.cursor.Refresh()
	prm, fg := w.getThemeColors()
	if w.recolorImage(prm, fg, w.imgProgressPixel) {
		w.img.Refresh()
	}
}

var _ desktop.Hoverable = (*WaveformSeekbar)(nil)

func (w *WaveformSeekbar) MouseIn(e *desktop.MouseEvent) {
	w.cursor.Move(fyne.NewPos(e.Position.X, 2))
	w.cursor.Show()
}

func (w *WaveformSeekbar) MouseMoved(e *desktop.MouseEvent) {
	w.cursor.Move(fyne.NewPos(e.Position.X, 2))
}

func (w *WaveformSeekbar) MouseOut() {
	w.cursor.Hide()
}

var _ fyne.Tappable = (*WaveformSeekbar)(nil)

func (w *WaveformSeekbar) Tapped(e *fyne.PointEvent) {
	if w.OnSeeked != nil {
		w.OnSeeked(float64(e.Position.X / w.Size().Width))
	}
}

func (w *WaveformSeekbar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.NewStack(
			container.New(layout.NewCustomPaddedLayout(4, 4, 0, 0), w.img),
			container.NewWithoutLayout(w.cursor),
		),
	)
}

// SetProgress sets how much of the seekbar has been played
// (ratio from 0 to 1)
func (w *WaveformSeekbar) SetProgress(v float64) {
	prm, fg := w.getThemeColors()
	thresholdPixel := int(math.Round(1024.0 /*pixel width of waveform*/ * v))
	if w.recolorImage(prm, fg, thresholdPixel) {
		w.img.Refresh()
	}
}

func (w *WaveformSeekbar) getThemeColors() (primary, foreground color.Color) {
	th := w.Theme()
	vnt := fyne.CurrentApp().Settings().ThemeVariant()
	primary = th.Color(theme.ColorNamePrimary, vnt)
	foreground = th.Color(theme.ColorNameForeground, vnt)
	return primary, foreground
}

func (w *WaveformSeekbar) recolorImage(cL, cR color.Color, progress int) (updated bool) {
	if w.img.Image == nil {
		return false
	}
	if w.imgColorL == cL && w.imgColorR == cR && w.imgProgressPixel == progress {
		return false
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
	for x := 0; x < bnds.Dx(); x++ {
		for y := 0; y < bnds.Dy(); y++ {
			if x < progress {
				setPixelRGB(img, x, y, rL, gL, bL)
			} else {
				setPixelRGB(img, x, y, rR, gR, bR)
			}
		}
	}

	w.imgColorL, w.imgColorR = cL, cR
	w.imgProgressPixel = progress
	return true
}

func setPixelRGB(img *image.NRGBA, x, y int, r, g, b byte) {
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = r
	img.Pix[offset+1] = g
	img.Pix[offset+2] = b
}
