package ui

import (
	"supersonic/backend"
	"supersonic/ui/widgets"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Page interface {
	fyne.Widget
}

type Searchable interface {
	OnSearched(string)
}

type CanPlayAlbum interface {
	SetPlayAlbumCallback(func(albumID string))
}

type BrowsingPane struct {
	widget.BaseWidget

	app *backend.App

	curPage Page

	forward    *widget.Button
	back       *widget.Button
	history    []Page
	historyIdx int

	searchBar         *widgets.SearchEntry
	pendingSearchLock sync.Mutex
	pendingSearch     bool
	searchGoroutine   bool

	container *fyne.Container
}

type blankPage struct {
	widget.Separator
}

func NewBrowsingPane(app *backend.App) *BrowsingPane {
	b := &BrowsingPane{app: app}
	b.ExtendBaseWidget(b)
	b.searchBar = widgets.NewSearchEntry()
	b.searchBar.OnTextChanged = b.onSearchTextChanged
	b.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), b.GoBack)
	b.forward = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), b.GoForward)
	b.curPage = &blankPage{}
	b.container = container.NewBorder(
		container.NewHBox(b.back, b.forward, b.searchBar),
		nil, nil, nil, b.curPage)
	return b
}

func (b *BrowsingPane) SetPage(p Page) {
	b.addPageToHistory(p)
	b.doSetPage(p)
}

func (b *BrowsingPane) doSetPage(p Page) {
	b.curPage = p
	if pa, ok := p.(CanPlayAlbum); ok {
		pa.SetPlayAlbumCallback(func(albumID string) {
			_ = b.app.PlaybackManager.PlayAlbum(albumID)
		})
	}
	_, s := p.(Searchable)
	b.searchBar.Hidden = !s
	b.container.Objects[0] = p
	b.Refresh()
}

func (b *BrowsingPane) addPageToHistory(p Page) {
	b.history = b.history[:b.historyIdx]
	b.history = append(b.history, p)
	b.historyIdx++
}

func (b *BrowsingPane) GoBack() {
	if b.historyIdx > 1 {
		b.historyIdx -= 1
		b.doSetPage(b.history[b.historyIdx-1])
	}
}

func (b *BrowsingPane) GoForward() {
	if b.historyIdx < len(b.history) {
		b.historyIdx++
		b.doSetPage(b.history[b.historyIdx-1])
	}
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
