package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SearchEntry struct {
	widget.Entry
	height        float32
	OnTextChanged func(string)
}

func NewSearchEntry() *SearchEntry {
	sf := &SearchEntry{}
	sf.ExtendBaseWidget(sf)
	// this is a bit hacky
	sf.height = widget.NewEntry().MinSize().Height
	sf.PlaceHolder = "Search"
	c := NewClearTextButton()
	c.OnTapped = func() {
		sf.SetText("")
	}
	sf.ActionItem = c
	sf.OnChanged = func(s string) {
		if s == "" {
			c.Resource = theme.SearchIcon()
		} else {
			c.Resource = theme.ContentClearIcon()
		}
		c.Refresh()
		if sf.OnTextChanged != nil {
			sf.OnTextChanged(s)
		}
	}
	return sf
}

func (s *SearchEntry) MinSize() fyne.Size {
	return fyne.NewSize(200, s.height)
}

var _ fyne.Tappable = (*clearTextButton)(nil)

type clearTextButton struct {
	widget.Icon

	OnTapped func()
}

func NewClearTextButton() *clearTextButton {
	c := &clearTextButton{}
	c.ExtendBaseWidget(c)
	c.Resource = theme.SearchIcon()
	return c
}

func (c *clearTextButton) Tapped(*fyne.PointEvent) {
	if c.OnTapped != nil {
		c.OnTapped()
	}
}
