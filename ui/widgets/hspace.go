package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type HSpace struct {
	widget.BaseWidget

	Width float32
}

func NewHSpace(w float32) *HSpace {
	h := &HSpace{Width: w}
	h.ExtendBaseWidget(h)
	return h
}

func (h *HSpace) MinSize() fyne.Size {
	return fyne.NewSize(h.Width, 0)
}

func (h *HSpace) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(layout.NewSpacer())
}
