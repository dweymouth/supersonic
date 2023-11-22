package widgets

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type MaxRowsLabel struct {
	widget.Label

	maxHeight float32
}

func NewMaxRowsLabel(maxRows int, text string) *MaxRowsLabel {
	if maxRows < 1 {
		maxRows = 1
	}
	m := &MaxRowsLabel{
		Label: widget.Label{
			Text: text,
		},
	}
	m.ExtendBaseWidget(m)

	maxHeightText := strings.Repeat("W\n", maxRows)
	maxHeightText = maxHeightText[:len(maxHeightText)-1]
	m.maxHeight = widget.NewLabel(maxHeightText).MinSize().Height

	return m
}

func (m *MaxRowsLabel) MinSize() fyne.Size {
	return fyne.NewSize(m.Label.MinSize().Width, m.maxHeight)
}
