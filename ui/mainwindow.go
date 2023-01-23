package ui

import (
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/browsing"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

var (
	ShortcutBack    = desktop.CustomShortcut{KeyName: fyne.KeyLeft, Modifier: AltModifier}
	ShortcutForward = desktop.CustomShortcut{KeyName: fyne.KeyRight, Modifier: AltModifier}
	ShortcutReload  = desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: ControlModifier}
	ShortcutSearch  = desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: ControlModifier}
)

type MainWindow struct {
	Window fyne.Window

	App          *backend.App
	Router       browsing.Router
	BrowsingPane *browsing.BrowsingPane
	BottomPanel  *BottomPanel

	container *fyne.Container
}

var (
	HomePage = browsing.AlbumsRoute(backend.AlbumSortRecentlyAdded)
)

func NewMainWindow(fyneApp fyne.App, appName string, app *backend.App, size fyne.Size) MainWindow {
	m := MainWindow{
		App:          app,
		Window:       fyneApp.NewWindow(appName),
		BrowsingPane: browsing.NewBrowsingPane(app),
	}

	m.Router = browsing.NewRouter(app, m.Window, m.BrowsingPane)
	m.BottomPanel = NewBottomPanel(app.Player, m.Router.OpenRoute)
	m.BottomPanel.SetPlaybackManager(app.PlaybackManager)
	m.BottomPanel.ImageManager = app.ImageManager
	m.container = container.NewBorder(nil, m.BottomPanel, nil, nil, m.BrowsingPane)
	m.Window.SetContent(m.container)
	m.Window.Resize(size)
	app.PlaybackManager.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			m.Window.SetTitle(appName)
			return
		}
		m.Window.SetTitle(song.Title)
	})
	app.ServerManager.OnServerConnected(func() {
		m.Router.OpenRoute(HomePage)
	})
	m.addNavigationButtons()
	m.addShortcuts()
	return m
}

func (m *MainWindow) PromptForFirstServer(cb func(string, string, string, string)) {
	d := widgets.NewAddServerForm("Connect to Server")
	pop := widget.NewModalPopUp(d, m.Canvas())
	d.OnSubmit = func() {
		pop.Hide()
		cb(d.Nickname, d.Host, d.Username, d.Password)
	}
	pop.Show()
}

func (m *MainWindow) addNavigationButtons() {
	m.BrowsingPane.AddNavigationButton(res.ResHeartFilledInvertPng, func() {
		m.Router.OpenRoute(browsing.FavoritesRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResDiscInvertPng, func() {
		m.Router.OpenRoute(browsing.AlbumsRoute(backend.AlbumSortRecentlyAdded))
	})
	m.BrowsingPane.AddNavigationButton(res.ResPeopleInvertPng, func() {
		m.Router.OpenRoute(browsing.ArtistsRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResTheatermasksInvertPng, func() {
		m.Router.OpenRoute(browsing.GenresRoute())
	})
}

func (m *MainWindow) addShortcuts() {
	m.Canvas().AddShortcut(&ShortcutBack, func(_ fyne.Shortcut) {
		m.BrowsingPane.GoBack()
		// TODO: reset focus only if something inside the page had focus
		m.Canvas().Focus(nil)
	})
	m.Canvas().AddShortcut(&ShortcutForward, func(_ fyne.Shortcut) {
		m.BrowsingPane.GoForward()
		m.Canvas().Focus(nil)
	})
	m.Canvas().AddShortcut(&ShortcutReload, func(_ fyne.Shortcut) {
		m.BrowsingPane.Reload()
	})
	m.Canvas().AddShortcut(&ShortcutSearch, func(_ fyne.Shortcut) {
		if s := m.BrowsingPane.GetSearchBarIfAny(); s != nil {
			m.Window.Canvas().Focus(s)
		}
	})

	m.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeySpace {
			m.App.Player.PlayPause()
		}
	})
}

func (m *MainWindow) Show() {
	m.Window.Show()
}

func (m *MainWindow) Canvas() fyne.Canvas {
	return m.Window.Canvas()
}

func (m *MainWindow) SetTitle(title string) {
	m.Window.SetTitle(title)
}

func (m *MainWindow) SetContent(c fyne.CanvasObject) {
	m.Window.SetContent(c)
}
