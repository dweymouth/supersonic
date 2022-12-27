package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	searchTimer         time.Timer
	pendingSearch       bool
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
	height float32
}

func NewSearchField() *searchField {
	sf := &searchField{}
	sf.ExtendBaseWidget(sf)
	// this is a bit hacky
	sf.height = widget.NewEntry().MinSize().Height
	sf.PlaceHolder = "Search"
	return sf
}

func (s *searchField) MinSize() fyne.Size {
	return fyne.NewSize(200, s.height)
}

func NewBrowsingPane() *BrowsingPane {
	b := &BrowsingPane{}
	b.ExtendBaseWidget(b)
	b.searchBar = NewSearchField()
	b.searchBar.OnChanged = b.onSearchTextChanged
	b.curPage = &blankPage{}
	b.searchTimer = *time.NewTimer(0)
	b.pageContaner = container.NewMax(b.curPage)
	b.container = container.NewBorder(container.NewHBox(b.searchBar), nil, nil, nil, b.pageContaner)
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
	b.searchTimer.Reset(300 * time.Millisecond)
	if s, ok := b.curPage.(Searchable); ok {
		s.OnSearched(text)
	}
	if !b.pendingSearch {
		/*
			ctx, cancel := context.WithCancel(context.Background())
			b.cancelPendingSearch = cancel
			b.pendingSearch = true
			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					b.pendingSearch = false
				case <-b.searchTimer.C:
					if s, ok := b.curPage.(Searchable); ok {
						s.OnSearched(b.searchBar.Text)
					}
					b.pendingSearch = false
				}
			}(ctx)
		*/
	}
}

func (b *BrowsingPane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}
