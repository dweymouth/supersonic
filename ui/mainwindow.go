package ui

import (
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/browsing"
	"supersonic/ui/controller"
	"supersonic/ui/os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/dweymouth/go-subsonic/subsonic"
)

var (
	ShortcutBack    = desktop.CustomShortcut{KeyName: fyne.KeyLeft, Modifier: os.AltModifier}
	ShortcutForward = desktop.CustomShortcut{KeyName: fyne.KeyRight, Modifier: os.AltModifier}
	ShortcutReload  = desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: os.ControlModifier}
	ShortcutSearch  = desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: os.ControlModifier}
)

type MainWindow struct {
	Window fyne.Window

	App          *backend.App
	Router       browsing.Router
	Controller   controller.Controller
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

	m.Controller = controller.Controller{
		MainWindow: m.Window,
		App:        app,
	}
	m.Router = browsing.NewRouter(app, m.Controller, m.BrowsingPane)
	m.BottomPanel = NewBottomPanel(app.Player, m.Router.OpenRoute)
	m.BottomPanel.SetPlaybackManager(app.PlaybackManager)
	m.BottomPanel.ImageManager = app.ImageManager
	m.container = container.NewBorder(nil, m.BottomPanel, nil, nil, m.BrowsingPane)
	m.Window.SetContent(m.container)
	m.Window.Resize(size)
	app.PlaybackManager.OnSongChange(func(song *subsonic.Child, _ *subsonic.Child) {
		if song == nil {
			m.Window.SetTitle(appName)
			return
		}
		m.Window.SetTitle(song.Title)
	})
	app.ServerManager.OnServerConnected(func() {
		m.BrowsingPane.EnableNavigationButtons()
		m.Router.OpenRoute(HomePage)
	})
	app.ServerManager.OnLogout(func() {
		m.BrowsingPane.DisableNavigationButtons()
		m.BrowsingPane.SetPage(nil)
		m.BrowsingPane.ClearHistory()
		m.Controller.PromptForLogin()
	})
	m.BrowsingPane.AddSettingsMenuItem("Log Out", app.ServerManager.Logout)
	m.addNavigationButtons()
	m.addShortcuts()
	return m
}

func (m *MainWindow) addNavigationButtons() {
	m.BrowsingPane.AddNavigationButton(res.ResHeadphonesInvertPng, func() {
		m.Router.OpenRoute(browsing.NowPlayingRoute())
	})
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
	m.BrowsingPane.AddNavigationButton(res.ResPlaylistInvertPng, func() {
		m.Router.OpenRoute(browsing.PlaylistsRoute())
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
	m.Canvas().AddShortcut(&fyne.ShortcutSelectAll{}, func(_ fyne.Shortcut) {
		m.BrowsingPane.SelectAll()
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
