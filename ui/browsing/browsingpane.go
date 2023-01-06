package browsing

import (
	"image/color"
	"supersonic/backend"
	"supersonic/ui/widgets"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type Page interface {
	fyne.CanvasObject

	Reload()
	Route() Route
}

type Searchable interface {
	OnSearched(string)
}

type CanPlayAlbum interface {
	SetPlayAlbumCallback(func(albumID string, startingTrack int))
}

type CanShowNowPlaying interface {
	OnSongChange(song *subsonic.Child)
}

type BrowsingPane struct {
	widget.BaseWidget

	app *backend.App

	curPage Page

	forward    *widget.Button
	back       *widget.Button
	reload     *widget.Button
	history    []Page
	historyIdx int

	searchBar         *widgets.SearchEntry
	pendingSearchLock sync.Mutex
	pendingSearch     bool
	searchGoroutine   bool

	pageContainer *fyne.Container
	container     *fyne.Container
}

func NewBrowsingPane(app *backend.App) *BrowsingPane {
	b := &BrowsingPane{app: app}
	b.ExtendBaseWidget(b)
	b.searchBar = widgets.NewSearchEntry()
	b.searchBar.OnTextChanged = b.onSearchTextChanged
	b.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), b.GoBack)
	b.forward = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), b.GoForward)
	b.reload = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), b.Reload)
	b.app.PlaybackManager.OnSongChange(b.onSongChange)
	b.pageContainer = container.NewMax(
		canvas.NewRectangle(color.RGBA{R: 24, G: 24, B: 24, A: 255}),
		layout.NewSpacer())
	b.container = container.NewBorder(
		container.NewHBox(b.back, b.forward, b.reload, b.searchBar),
		nil, nil, nil, b.pageContainer)
	return b
}

func (b *BrowsingPane) SetPage(p Page) {
	if b.doSetPage(p) {
		b.addPageToHistory(p)
	}
}

func (b *BrowsingPane) doSetPage(p Page) bool {
	if b.curPage != nil && b.curPage.Route() == p.Route() {
		return false
	}
	b.curPage = p
	if pa, ok := p.(CanPlayAlbum); ok {
		pa.SetPlayAlbumCallback(func(albumID string, firstTrack int) {
			_ = b.app.PlaybackManager.PlayAlbum(albumID, firstTrack)
		})
	}
	if np, ok := p.(CanShowNowPlaying); ok {
		np.OnSongChange(b.app.PlaybackManager.NowPlaying())
	}
	_, s := p.(Searchable)
	b.searchBar.Hidden = !s
	b.pageContainer.Objects[1] = p
	b.Refresh()
	return true
}

func (b *BrowsingPane) onSongChange(song *subsonic.Child) {
	if b.curPage == nil {
		return
	}
	if p, ok := b.curPage.(CanShowNowPlaying); ok {
		p.OnSongChange(song)
	}
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

func (b *BrowsingPane) Reload() {
	if b.curPage != nil {
		b.curPage.Reload()
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
