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

type ImagePlaceholder struct {
	widget.BaseWidget
	container *fyne.Container
	minSize   float32
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

func (i *ImagePlaceholder) SetImage(img image.Image) {
	cImg := canvas.NewImageFromImage(img)
	cImg.FillMode = canvas.ImageFillContain
	cImg.SetMinSize(fyne.NewSize(i.minSize, i.minSize))
	i.container.RemoveAll()
	i.container.Add(cImg)
	i.container.Refresh()
}

func (i *ImagePlaceholder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.container)
}
