package widgets

import (
	"supersonic/res"
	"supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	themedResStarFilled  = theme.NewThemedResource(res.ResStarFilledSvg)
	themedResStarOutline = theme.NewThemedResource(res.ResStarOutlineSvg)
)

type StarRating struct {
	widget.BaseWidget

	Rating   int
	StarSize float32

	container *fyne.Container
}

func NewStarRating() *StarRating {
	s := &StarRating{}
	s.ExtendBaseWidget(s)
	return s
}

func (s *StarRating) createContainer() {
	s.container = container.New(&layouts.HboxCustomPadding{
		DisableThemePad: true,
		ExtraPad:        2,
	})
	var im *canvas.Image
	for i := 0; i < 5; i++ {
		if s.Rating > i {
			im = canvas.NewImageFromResource(themedResStarFilled)
		} else {
			im = canvas.NewImageFromResource(themedResStarOutline)
		}
		im.SetMinSize(fyne.NewSize(s.StarSize, s.StarSize))
		s.container.Add(im)
	}
}

func (s *StarRating) Refresh() {
	// widget has not had renderer created yet
	if s.container == nil {
		return
	}
	for i := 0; i < 5; i++ {
		im := s.container.Objects[i].(*canvas.Image)
		im.SetMinSize(fyne.NewSize(s.StarSize, s.StarSize))
		if s.Rating > i {
			im.Resource = themedResStarFilled
		} else {
			im.Resource = themedResStarOutline
		}
	}
	s.BaseWidget.Refresh()
}

func (s *StarRating) MinSize() fyne.Size {
	return fyne.NewSize(s.StarSize*5, s.StarSize)
}

func (s *StarRating) CreateRenderer() fyne.WidgetRenderer {
	if s.container == nil {
		s.createContainer()
	}
	return widget.NewSimpleRenderer(container.NewCenter(s.container))
}
