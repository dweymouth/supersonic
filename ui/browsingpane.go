package ui

import (
	"supersonic/ui/widgets"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Page interface {
	fyne.Widget
}

type Searchable interface {
	OnSearched(string)
}

type BrowsingPane struct {
	widget.BaseWidget

	searchBar         *widgets.SearchEntry
	pendingSearchLock sync.Mutex
	pendingSearch     bool
	searchGoroutine   bool
	curPage           Page

	container *fyne.Container
}

type blankPage struct {
	widget.Separator
}

type hspace struct {
	widget.BaseWidget

	Width float32
}

func newHSpace(w float32) *hspace {
	h := &hspace{Width: w}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hspace) MinSize() fyne.Size {
	return fyne.NewSize(h.Width, 0)
}

func (h *hspace) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(layout.NewSpacer())
}

func NewBrowsingPane() *BrowsingPane {
	b := &BrowsingPane{}
	b.ExtendBaseWidget(b)
	b.searchBar = widgets.NewSearchEntry()
	b.searchBar.OnTextChanged = b.onSearchTextChanged
	b.curPage = &blankPage{}
	b.container = container.NewBorder(
		container.NewHBox(newHSpace(15), b.searchBar),
		nil, nil, nil, b.curPage)
	return b
}

func (b *BrowsingPane) SetPage(p Page) {
	b.curPage = p
	b.container.Objects[0] = p
	b.Refresh()
}

func (b *BrowsingPane) onSearchTextChanged(text string) {
	if text == "" {
		b.sendSearch("")
		return
	}
	b.pendingSearchLock.Lock()
	defer b.pendingSearchLock.Unlock()
	b.pendingSearch = true
	if !b.searchGoroutine {
		go b.waitAndSearch()
		b.searchGoroutine = true
	}
}

func (b *BrowsingPane) waitAndSearch() {
	t := time.NewTicker(200 * time.Millisecond)
	var getReadyToSearch bool
	var done bool
	for !done {
		<-t.C
		b.pendingSearchLock.Lock()
		if b.pendingSearch {
			getReadyToSearch = true
			b.pendingSearch = false
		} else if getReadyToSearch {
			b.sendSearch(b.searchBar.Text)
			t.Stop()
			b.searchGoroutine = false
			done = true
		}
		b.pendingSearchLock.Unlock()
	}
}

func (b *BrowsingPane) sendSearch(query string) {
	if s, ok := b.curPage.(Searchable); ok {
		s.OnSearched(query)
	}
}

func (b *BrowsingPane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}
