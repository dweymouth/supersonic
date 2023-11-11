package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

type Carousel struct {
	widget.BaseWidget

	Title string
	Items []GridViewItemModel

	titleDisplay *widget.Label
}

func NewCarousel(title string, items []GridViewItemModel) *Carousel {
	c := &Carousel{Title: title, Items: items}
	c.ExtendBaseWidget(c)
	c.titleDisplay = widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return c
}

func (c *Carousel) CreateRenderer() fyne.WidgetRenderer {
	bg := myTheme.NewThemedRectangle(myTheme.ColorNamePageHeader)
	bg.CornerRadiusName = theme.SizeNameInputRadius
	container := container.NewStack(bg)
	return widget.NewSimpleRenderer(container)
}
