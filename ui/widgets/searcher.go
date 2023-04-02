package widgets

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Searcher is a search entry widget that will issue a search command
// (aka call OnSearched) when a short span of time has elapsed since
// the user typed into the widget.
type Searcher struct {
	Entry      *SearchEntry
	OnSearched func(string)

	searchGoroutine   bool
	pendingSearch     bool
	pendingSearchLock sync.Mutex
}

func NewSearcher() *Searcher {
	s := &Searcher{
		Entry: NewSearchEntry(),
	}
	s.Entry.OnChanged = s.onSearchTextChanged
	return s
}

func (s *Searcher) onSearchTextChanged(text string) {
	if text == "" {
		s.sendSearch("")
		return
	}
	s.pendingSearchLock.Lock()
	defer s.pendingSearchLock.Unlock()
	s.pendingSearch = true
	if !s.searchGoroutine {
		go s.waitAndSearch()
		s.searchGoroutine = true
	}
}

func (s *Searcher) waitAndSearch() {
	t := time.NewTicker(200 * time.Millisecond)
	var getReadyToSearch bool
	var done bool
	for !done {
		<-t.C
		s.pendingSearchLock.Lock()
		if s.pendingSearch {
			getReadyToSearch = true
			s.pendingSearch = false
		} else if getReadyToSearch {
			s.sendSearch(s.Entry.Text)
			t.Stop()
			s.searchGoroutine = false
			done = true
		}
		s.pendingSearchLock.Unlock()
	}
}

func (s *Searcher) sendSearch(text string) {
	if s.OnSearched != nil {
		s.OnSearched(text)
	}
}

type SearchEntry struct {
	widget.Entry
}

func NewSearchEntry() *SearchEntry {
	sf := &SearchEntry{}
	sf.ExtendBaseWidget(sf)
	sf.PlaceHolder = "Search"
	sf.ActionItem = NewClearTextButton(func() {
		sf.SetText("")
	})
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
