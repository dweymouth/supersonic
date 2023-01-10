package browsing

import (
	"image/color"
	"supersonic/backend"

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

type CanPlayAlbum interface {
	SetPlayAlbumCallback(func(albumID string, startingTrack int))
}

type CanShowNowPlaying interface {
	OnSongChange(song *subsonic.Child)
}

type BrowsingPane struct {
	widget.BaseWidget

	app *backend.App

	// Invoked when the home button is clicked
	OnGoHome func()

	curPage Page

	home       *widget.Button
	forward    *widget.Button
	back       *widget.Button
	reload     *widget.Button
	history    []Page
	historyIdx int

	pageContainer *fyne.Container
	container     *fyne.Container
}

func NewBrowsingPane(app *backend.App) *BrowsingPane {
	b := &BrowsingPane{app: app}
	b.ExtendBaseWidget(b)
	b.home = widget.NewButtonWithIcon("", theme.HomeIcon(), b.goHome)
	b.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), b.GoBack)
	b.forward = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), b.GoForward)
	b.reload = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), b.Reload)
	b.app.PlaybackManager.OnSongChange(b.onSongChange)
	b.pageContainer = container.NewMax(
		canvas.NewRectangle(color.RGBA{R: 24, G: 24, B: 24, A: 255}),
		layout.NewSpacer())
	b.container = container.NewBorder(
		container.NewHBox(b.home, b.back, b.forward, b.reload),
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
	// allow garbage collection of pages that will be removed from the history
	for i := b.historyIdx; i < len(b.history); i++ {
		b.history[i] = nil
	}
	b.history = b.history[:b.historyIdx]
	b.history = append(b.history, p)
	b.historyIdx++
}

func (b *BrowsingPane) goHome() {
	if b.OnGoHome != nil {
		b.OnGoHome()
	}
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

func (b *BrowsingPane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}
