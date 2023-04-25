package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type hyperlinkWrapper struct {
	widget.Hyperlink

	textWidthCached float32
	MaxWidth        float32
}

func newHyperlinkWrapper() *hyperlinkWrapper {
	h := &hyperlinkWrapper{
		Hyperlink: widget.Hyperlink{
			Text:     "",
			Wrapping: fyne.TextTruncate,
		},
		textWidthCached: -1,
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hyperlinkWrapper) MinSize() fyne.Size {
	if h.textWidthCached < 0 {
		s := fyne.MeasureText(h.Text, theme.TextSize(), h.TextStyle)
		// the 2.7 factor is a bit of a magic number but it works ¯\_(ツ)_/¯
		h.textWidthCached = s.Width + theme.Padding()*2.7
	}
	return fyne.NewSize(fyne.Min(h.MaxWidth, h.textWidthCached), h.Hyperlink.MinSize().Height)
}

func (h *hyperlinkWrapper) SetText(text string) {
	h.Text = text
	h.textWidthCached = -1
}

func (h *hyperlinkWrapper) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeySpace {
		if h.OnTapped != nil {
			h.OnTapped()
		}
	}
}

type CustomHyperlink struct {
	widget.BaseWidget
	h *hyperlinkWrapper
	l *widget.Label

	OnTapped   func()
	NoTruncate bool
	Disabled   bool

	lastDisabled bool
	container    *fyne.Container
	minSize      fyne.Size
}

func NewCustomHyperlink() *CustomHyperlink {
	c := &CustomHyperlink{
		h:         newHyperlinkWrapper(),
		l:         widget.NewLabel(""),
		container: container.NewMax(),
	}
	c.h.OnTapped = func() {
		if c.OnTapped != nil {
			c.OnTapped()
		}
	}
	c.ExtendBaseWidget(c)
	c.minSize = c.h.MinSize()
	c.updateContainer(c.Disabled)
	return c
}

func (c *CustomHyperlink) SetText(text string) {
	c.l.Text = text
	lastWrapping := c.l.Wrapping
	c.l.Wrapping = fyne.TextWrapOff
	s := c.l.MinSize()
	c.h.SetText(text)
	if c.NoTruncate {
		c.minSize = s
	} else {
		c.minSize = fyne.NewSize(fyne.Min(c.Size().Width, s.Width), s.Height)
	}
	c.l.Wrapping = lastWrapping
	c.Refresh()
}

func (c *CustomHyperlink) Resize(size fyne.Size) {
	c.h.MaxWidth = size.Width
	c.BaseWidget.Resize(size)
}

func (c *CustomHyperlink) Refresh() {
	if c.NoTruncate {
		c.l.Wrapping = fyne.TextWrapOff
	} else {
		c.l.Wrapping = fyne.TextTruncate
	}
	if c.lastDisabled != c.Disabled {
		c.updateContainer(c.Disabled)
		c.lastDisabled = c.Disabled
	}
	c.container.Refresh()
}

func (c *CustomHyperlink) MinSize() fyne.Size {
	return c.minSize
}

func (c *CustomHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

func (c *CustomHyperlink) updateContainer(linkDisabled bool) {
	c.container.RemoveAll()
	if linkDisabled {
		c.container.Add(c.l)
	} else {
		c.container.Add(container.NewHBox(c.h, layout.NewSpacer()))
	}
}
