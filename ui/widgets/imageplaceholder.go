package widgets

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

// A widget that can display an image or else
// a placeholder with a rectangular border frame
// and an icon positioned in the center of the frame.
type ImagePlaceholder struct {
	ScaleMode canvas.ImageScale

	widget.BaseWidget
	content   *fyne.Container
	imageDisp *TappableImage
	image     image.Image
	iconImage *canvas.Image
	border    *myTheme.ThemedRectangle
	minSize   float32

	OnTapped          func(*fyne.PointEvent)
	OnTappedSecondary func(*fyne.PointEvent)
}

func NewImagePlaceholder(centerIcon fyne.Resource, minSize float32) *ImagePlaceholder {
	i := &ImagePlaceholder{minSize: minSize}
	i.ExtendBaseWidget(i)
	i.iconImage = canvas.NewImageFromResource(centerIcon)
	i.iconImage.FillMode = canvas.ImageFillContain
	i.iconImage.SetMinSize(fyne.NewSize(minSize/4, minSize/4))
	i.imageDisp = NewTappableImage(i.onTapped)
	i.imageDisp.OnTappedSecondary = i.onTappedSecondary
	i.imageDisp.FillMode = canvas.ImageFillContain
	i.imageDisp.Hidden = true
	i.border = myTheme.NewThemedRectangle(theme.ColorNameBackground)
	i.border.BorderColorName = theme.ColorNameForeground
	i.border.BorderWidth = 3
	i.content = container.NewMax(
		i.border,
		container.NewCenter(i.iconImage),
		i.imageDisp,
	)
	return i
}

func (i *ImagePlaceholder) HaveImage() bool {
	return i.image != nil
}

func (i *ImagePlaceholder) SetImage(img image.Image, tappable bool) {
	i.image = img
	if img != nil {
		i.imageDisp.DisableTapping = !tappable
		i.imageDisp.Image.Image = img
	}

	i.Refresh()
}

func (i *ImagePlaceholder) Image() image.Image {
	return i.image
}

func (i *ImagePlaceholder) onTapped(e *fyne.PointEvent) {
	if i.OnTapped != nil {
		i.OnTapped(e)
	}
}

func (i *ImagePlaceholder) onTappedSecondary(e *fyne.PointEvent) {
	if i.OnTappedSecondary != nil {
		i.OnTappedSecondary(e)
	}
}

func (i *ImagePlaceholder) MinSize() fyne.Size {
	return fyne.NewSize(i.minSize, i.minSize)
}

func (i *ImagePlaceholder) Refresh() {
	i.border.Hidden = i.HaveImage()
	i.iconImage.Hidden = i.HaveImage()
	i.imageDisp.Hidden = !i.HaveImage()
	i.imageDisp.ScaleMode = i.ScaleMode
	i.iconImage.ScaleMode = i.ScaleMode
	i.BaseWidget.Refresh()
}

func (i *ImagePlaceholder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.content)
}
