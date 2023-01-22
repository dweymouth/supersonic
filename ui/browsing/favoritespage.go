package browsing

import (
	"log"
	"supersonic/backend"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type FavoritesPage struct {
	widget.BaseWidget

	im        *backend.ImageManager
	sm        *backend.ServerManager
	nav       func(Route)
	grid      *widgets.AlbumGrid
	titleDisp *widget.RichText
	container *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewFavoritesPage(sm *backend.ServerManager, im *backend.ImageManager, nav func(Route)) *FavoritesPage {
	a := &FavoritesPage{
		sm:  sm,
		im:  im,
		nav: nav,
	}
	a.ExtendBaseWidget(a)
	a.titleDisp = widget.NewRichTextWithText("Favorites")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.container = container.NewBorder(
		container.NewHBox(widgets.NewHSpace(9), a.titleDisp), nil, nil, nil, layout.NewSpacer())
	a.loadAsync()
	return a
}

func (a *FavoritesPage) Route() Route {
	return FavoritesRoute()
}

func (a *FavoritesPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *FavoritesPage) Reload() {
	a.loadAsync()
}

func (a *FavoritesPage) Save() SavedPage {
	return &savedFavoritesPage{
		sm:  a.sm,
		im:  a.im,
		nav: a.nav,
	}
}

func (a *FavoritesPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID, 0)
	}
}

func (a *FavoritesPage) onShowAlbumPage(albumID string) {
	a.nav(AlbumRoute(albumID))
}

func (a *FavoritesPage) loadAsync() {
	go func() {
		starred, err := a.sm.Server.GetStarred2(nil)
		if err != nil {
			log.Printf("Failed to get favorites: %s", err.Error())
			return
		}
		ag := widgets.NewFixedAlbumGrid(starred.Album, a.im, true /*showYear*/)
		ag.OnPlayAlbum = a.onPlayAlbum
		ag.OnShowAlbumPage = a.onShowAlbumPage
		a.container.Objects[0] = ag
		a.container.Refresh()
	}()
}

func (a *FavoritesPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedFavoritesPage struct {
	artistID string
	sm       *backend.ServerManager
	im       *backend.ImageManager
	nav      func(Route)
}

func (s *savedFavoritesPage) Restore() Page {
	return NewFavoritesPage(s.sm, s.im, s.nav)
}
