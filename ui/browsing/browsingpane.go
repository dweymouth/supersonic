package browsing

import (
	"image/color"
	"supersonic/backend"
	"supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type Page interface {
	fyne.CanvasObject

	Save() SavedPage
	Reload()
	Route() Route
}

type SavedPage interface {
	Restore() Page
}

// Searchable pages should implement this interface so their search bar can be focused by keyboard shortcut.
type Searchable interface {
	SearchWidget() fyne.Focusable
}

type CanSelectAll interface {
	SelectAll()
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
	history    []SavedPage
	historyIdx int

	navBtnsContainer *fyne.Container
	pageContainer    *fyne.Container
	container        *fyne.Container
}

func NewBrowsingPane(app *backend.App) *BrowsingPane {
	b := &BrowsingPane{app: app}
	b.ExtendBaseWidget(b)
	b.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), b.GoBack)
	b.forward = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), b.GoForward)
	b.reload = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), b.Reload)
	b.app.PlaybackManager.OnSongChange(b.onSongChange)
	b.pageContainer = container.NewMax(
		canvas.NewRectangle(color.RGBA{R: 24, G: 24, B: 24, A: 255}),
		layout.NewSpacer())
	b.navBtnsContainer = container.NewHBox()
	b.container = container.NewBorder(
		container.New(layouts.NewLeftMiddleRightLayout(0),
			container.NewHBox(b.back, b.forward, b.reload), b.navBtnsContainer, layout.NewSpacer()),
		nil, nil, nil, b.pageContainer)
	return b
}

func (b *BrowsingPane) SetPage(p Page) {
	oldPage := b.curPage
	if b.doSetPage(p) && oldPage != nil {
		b.addPageToHistory(oldPage, true)
	}
}

func (b *BrowsingPane) AddNavigationButton(iconRes fyne.Resource, action func()) {
	b.navBtnsContainer.Add(widget.NewButtonWithIcon("", iconRes, action))
}

func (b *BrowsingPane) GetSearchBarIfAny() fyne.Focusable {
	if s, ok := b.curPage.(Searchable); ok {
		return s.SearchWidget()
	}
	return nil
}

func (b *BrowsingPane) SelectAll() {
	if s, ok := b.curPage.(CanSelectAll); ok {
		s.SelectAll()
	}
}

func (b *BrowsingPane) doSetPage(p Page) bool {
	if b.curPage != nil && b.curPage.Route() == p.Route() {
		return false
	}
	b.curPage = p
	if np, ok := p.(CanShowNowPlaying); ok {
		np.OnSongChange(b.app.PlaybackManager.NowPlaying())
	}
	b.pageContainer.Remove(b.curPage)
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

func (b *BrowsingPane) addPageToHistory(p Page, truncate bool) {
	if truncate {
		// allow garbage collection of pages that will be removed from the history
		for i := b.historyIdx; i < len(b.history); i++ {
			b.history[i] = nil
		}
		b.history = b.history[:b.historyIdx]
	}
	if b.historyIdx < len(b.history) {
		b.history[b.historyIdx] = p.Save()
	} else {
		b.history = append(b.history, p.Save())
	}
	b.historyIdx++
}

func (b *BrowsingPane) GoBack() {
	if b.historyIdx > 0 {
		b.addPageToHistory(b.curPage, false)
		b.historyIdx -= 2
		b.doSetPage(b.history[b.historyIdx].Restore())
	}
}

func (b *BrowsingPane) GoForward() {
	if b.historyIdx < len(b.history)-1 {
		b.addPageToHistory(b.curPage, false)
		b.doSetPage(b.history[b.historyIdx].Restore())
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
