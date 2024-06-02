package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func NewInfoMessage(title, subtitle string) fyne.CanvasObject {
	c := container.New(layout.NewCustomPaddedVBoxLayout(-10),
		container.NewCenter(
			container.NewBorder(nil, nil,
				widget.NewIcon(theme.InfoIcon()), nil,
				widget.NewRichText(&widget.TextSegment{
					Text:  title,
					Style: widget.RichTextStyleSubHeading,
				}))),
	)
	if subtitle != "" {
		c.Add(container.NewCenter(widget.NewLabel(subtitle)))
	}
	return c
}
