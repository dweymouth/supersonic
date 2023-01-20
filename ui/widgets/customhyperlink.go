package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type hyperlinkWrapper struct {
	widget.Hyperlink

	l        *widget.Label
	MaxWidth float32
}

func newHyperlinkWrapper() *hyperlinkWrapper {
	h := &hyperlinkWrapper{
		Hyperlink: widget.Hyperlink{
			Text:     "",
			Wrapping: fyne.TextTruncate,
		},
		l: widget.NewLabel(""),
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hyperlinkWrapper) MinSize() fyne.Size {
	w := fyne.Min(h.MaxWidth, h.l.MinSize().Width)
	return fyne.NewSize(w, h.Hyperlink.MinSize().Height)
}

func (h *hyperlinkWrapper) SetText(text string) {
	h.Text = text
	h.l.SetText(text)
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

	OnTapped func()

	container *fyne.Container
	minSize   fyne.Size
}

func NewCustomHyperlink() *CustomHyperlink {
	c := &CustomHyperlink{
		h: newHyperlinkWrapper(),
	}
	c.h.OnTapped = func() {
		if c.OnTapped != nil {
			c.OnTapped()
		}
	}
	c.ExtendBaseWidget(c)
	c.container = container.NewHBox(c.h, layout.NewSpacer())
	return c
}

func (c *CustomHyperlink) SetText(text string) {
	s := widget.NewLabel(text).MinSize()
	c.h.SetText(text)
	c.minSize = fyne.NewSize(fyne.Min(c.Size().Width, s.Width), s.Height)
	c.Refresh()
}

func (c *CustomHyperlink) Resize(size fyne.Size) {
	c.h.MaxWidth = size.Width
	c.BaseWidget.Resize(size)
}

func (c *CustomHyperlink) Refresh() {
	c.container.Refresh()
}

func (c *CustomHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}
