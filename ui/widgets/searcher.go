package widgets

import (
	"sync"
	"time"
)

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
	s.Entry.OnTextChanged = s.onSearchTextChanged
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
