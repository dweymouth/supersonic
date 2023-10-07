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
		// the 4.0 factor is a bit of a magic number but it works ¯\_(ツ)_/¯
		h.textWidthCached = s.Width + theme.Padding()*4.0
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

	lastDisabled  bool
	container     *fyne.Container
	fullTextWidth float32
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
	c.fullTextWidth = c.l.MinSize().Width
	c.ExtendBaseWidget(c)
	c.updateContainer(c.Disabled)
	return c
}

func (c *CustomHyperlink) SetText(text string) {
	lastWrapping := c.l.Wrapping
	c.l.Wrapping = fyne.TextWrapOff
	c.l.SetText(text)
	c.fullTextWidth = c.l.MinSize().Width
	c.h.SetText(text)
	c.l.Wrapping = lastWrapping
	c.Refresh()
}

func (c *CustomHyperlink) SetTextStyle(style fyne.TextStyle) {
	c.h.TextStyle = style
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
	if c.NoTruncate {
		return fyne.NewSize(c.fullTextWidth, c.l.MinSize().Height)
	}
	return fyne.NewSize(0, c.l.MinSize().Height)
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
