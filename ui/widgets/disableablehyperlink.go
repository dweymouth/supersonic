package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type DisableableHyperlink struct {
	widget.BaseWidget
	h *widget.Hyperlink
	l *widget.Label

	OnTapped   func()
	NoTruncate bool
	Disabled   bool

	lastDisabled bool
	container    *fyne.Container
}

func NewDisableableHyperlink() *DisableableHyperlink {
	c := &DisableableHyperlink{
		h:         widget.NewHyperlink("", nil),
		l:         widget.NewLabel(""),
		container: container.NewMax(),
	}
	c.h.OnTapped = func() {
		if c.OnTapped != nil {
			c.OnTapped()
		}
	}
	c.ExtendBaseWidget(c)
	c.updateContainer(c.Disabled)
	return c
}

func (c *DisableableHyperlink) SetText(text string) {
	lastWrapping := c.l.Wrapping
	c.l.Wrapping = fyne.TextWrapOff
	c.l.SetText(text)
	c.h.SetText(text)
	c.l.Wrapping = lastWrapping
	c.Refresh()
}

func (c *DisableableHyperlink) Refresh() {
	if c.NoTruncate {
		c.h.Wrapping = fyne.TextWrapOff
		c.l.Wrapping = fyne.TextWrapOff
	} else {
		c.h.Wrapping = fyne.TextTruncate
		c.l.Wrapping = fyne.TextTruncate
	}
	if c.lastDisabled != c.Disabled {
		c.updateContainer(c.Disabled)
		c.lastDisabled = c.Disabled
	}
	c.container.Refresh()
}

func (c *DisableableHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

func (c *DisableableHyperlink) updateContainer(linkDisabled bool) {
	c.container.RemoveAll()
	if linkDisabled {
		c.container.Add(c.l)
	} else {
		c.container.Add(c.h)
	}
}
