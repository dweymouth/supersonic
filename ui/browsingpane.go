package ui

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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

	searchBar           *searchField
	pendingSearchLock   sync.Mutex
	pendingSearch       bool
	searchGoroutine     bool
	cancelPendingSearch context.CancelFunc
	curPage             Page

	container    *fyne.Container
	pageContaner *fyne.Container
}

type blankPage struct {
	widget.Separator
}

type searchField struct {
	widget.Entry
	height        float32
	OnTextChanged func(string)
}

var _ fyne.Tappable = (*clearTextButton)(nil)

type clearTextButton struct {
	widget.Icon

	OnTapped func()
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

func NewSearchField() *searchField {
	sf := &searchField{}
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

func (s *searchField) MinSize() fyne.Size {
	return fyne.NewSize(200, s.height)
}

func NewBrowsingPane() *BrowsingPane {
	b := &BrowsingPane{}
	b.ExtendBaseWidget(b)
	b.searchBar = NewSearchField()
	b.searchBar.OnTextChanged = b.onSearchTextChanged
	b.curPage = &blankPage{}
	b.pageContaner = container.NewMax(b.curPage)
	b.container = container.NewBorder(
		container.NewHBox(newHSpace(15), b.searchBar),
		nil, nil, nil, b.pageContaner)
	return b
}

func (b *BrowsingPane) SetPage(p Page) {
	if b.cancelPendingSearch != nil {
		b.cancelPendingSearch()
		b.cancelPendingSearch = nil
	}
	b.curPage = p
	b.pageContaner.Objects[0] = p
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
		select {
		case <-t.C:
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
}

func (b *BrowsingPane) sendSearch(query string) {
	if s, ok := b.curPage.(Searchable); ok {
		s.OnSearched(query)
	}
}

func (b *BrowsingPane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}
