package widgets

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/ui/util"
)

// SearchEntry is a search entry widget that will issue a search command
// (aka call OnSearched) when a short span of time has elapsed since
// the user typed into the widget.
type SearchEntry struct {
	widget.Entry

	OnSearched func(string)
}

func NewSearchEntry() *SearchEntry {
	sf := &SearchEntry{}
	sf.ExtendBaseWidget(sf)
	sf.PlaceHolder = "Search"
	sf.ActionItem = NewClearTextButton(func() {
		sf.SetText("")
	})
	debounceFunc := util.NewDebouncer(200*time.Millisecond, func() {
		sf.sendSearch(sf.Entry.Text)
	})
	sf.Entry.OnChanged = func(_ string) { debounceFunc() }
	return sf
}

func (s *SearchEntry) Refresh() {
	if s.Text == "" {
		s.ActionItem.(*clearTextButton).Resource = theme.SearchIcon()
	} else {
		s.ActionItem.(*clearTextButton).Resource = theme.ContentClearIcon()
	}
	s.Entry.Refresh()
}

func (s *SearchEntry) MinSize() fyne.Size {
	return fyne.NewSize(200, s.Entry.MinSize().Height)
}

var _ fyne.Tappable = (*clearTextButton)(nil)

type clearTextButton struct {
	widget.Icon

	OnTapped func()
}

func (s *SearchEntry) sendSearch(text string) {
	if s.OnSearched != nil {
		s.OnSearched(text)
	}
}

func NewClearTextButton(onTapped func()) *clearTextButton {
	c := &clearTextButton{OnTapped: onTapped}
	c.ExtendBaseWidget(c)
	c.Resource = theme.SearchIcon()
	return c
}

func (c *clearTextButton) Tapped(*fyne.PointEvent) {
	if c.OnTapped != nil {
		c.OnTapped()
	}
}
