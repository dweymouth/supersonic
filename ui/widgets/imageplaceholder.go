package widgets

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// A widget that can display an image or else
// a placeholder with a rectangular border frame
// and an icon positioned in the center of the frame.
type ImagePlaceholder struct {
	widget.BaseWidget
	container *fyne.Container
	image     image.Image
	minSize   float32

	OnTapped func()
}

func NewImagePlaceholder(centerIcon fyne.Resource, minSize float32) *ImagePlaceholder {
	m := &ImagePlaceholder{minSize: minSize}
	m.ExtendBaseWidget(m)
	img := canvas.NewImageFromResource(centerIcon)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(minSize/4, minSize/4))
	rect := canvas.NewRectangle(theme.BackgroundColor())
	rect.StrokeColor = color.Black
	rect.StrokeWidth = 3
	rect.SetMinSize(fyne.NewSize(minSize, minSize))
	m.container = container.NewMax(
		rect,
		container.NewCenter(img),
	)
	return m
}

type CanvasImage interface {
	fyne.CanvasObject

	SetMinSize(fyne.Size)
}

func (i *ImagePlaceholder) HaveImage() bool {
	return i.image != nil
}

func (i *ImagePlaceholder) SetImage(img image.Image, tappable bool) {
	var cIm CanvasImage
	i.image = img
	if tappable {
		cImg := NewTappableImage(i.onTapped)
		cImg.Image.Image = img
		cImg.FillMode = canvas.ImageFillContain
		cIm = cImg
	} else {
		cImg := canvas.NewImageFromImage(img)
		cImg.FillMode = canvas.ImageFillContain
		cIm = cImg
	}
	cIm.SetMinSize(fyne.NewSize(i.minSize, i.minSize))
	i.container.RemoveAll()
	i.container.Add(cIm)
	i.container.Refresh()
}

func (i *ImagePlaceholder) Image() image.Image {
	return i.image
}

func (i *ImagePlaceholder) onTapped() {
	if i.OnTapped != nil {
		i.OnTapped()
	}
}

func (i *ImagePlaceholder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.container)
}
