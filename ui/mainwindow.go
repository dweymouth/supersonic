package ui

import (
	"fmt"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/browsing"
	"supersonic/ui/controller"
	"supersonic/ui/os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

var (
	ShortcutReload      = desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: os.ControlModifier}
	ShortcutSearch      = desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: os.ControlModifier}
	ShortcutCloseWindow = desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: os.ControlModifier}

	ShortcutNavOne   = desktop.CustomShortcut{KeyName: fyne.Key1, Modifier: os.ControlModifier}
	ShortcutNavTwo   = desktop.CustomShortcut{KeyName: fyne.Key2, Modifier: os.ControlModifier}
	ShortcutNavThree = desktop.CustomShortcut{KeyName: fyne.Key3, Modifier: os.ControlModifier}
	ShortcutNavFour  = desktop.CustomShortcut{KeyName: fyne.Key4, Modifier: os.ControlModifier}
	ShortcutNavFive  = desktop.CustomShortcut{KeyName: fyne.Key5, Modifier: os.ControlModifier}
	ShortcutNavSix   = desktop.CustomShortcut{KeyName: fyne.Key6, Modifier: os.ControlModifier}
	ShortcutNavSeven = desktop.CustomShortcut{KeyName: fyne.Key7, Modifier: os.ControlModifier}

	NavShortcuts = []desktop.CustomShortcut{ShortcutNavOne, ShortcutNavTwo, ShortcutNavThree,
		ShortcutNavFour, ShortcutNavFive, ShortcutNavSix, ShortcutNavSeven}
)

type MainWindow struct {
	Window fyne.Window

	App          *backend.App
	Router       browsing.Router
	Controller   *controller.Controller
	BrowsingPane *browsing.BrowsingPane
	BottomPanel  *BottomPanel

	haveSystemTray bool
	container      *fyne.Container
}

func NewMainWindow(fyneApp fyne.App, appName, appVersion string, app *backend.App, size fyne.Size) MainWindow {
	m := MainWindow{
		App:          app,
		Window:       fyneApp.NewWindow(appName),
		BrowsingPane: browsing.NewBrowsingPane(app),
	}

	if app.Config.Application.EnableSystemTray {
		m.SetupSystemTrayMenu(appName, fyneApp)
	}
	m.Controller = &controller.Controller{
		AppVersion: appVersion,
		MainWindow: m.Window,
		App:        app,
	}
	m.Router = browsing.NewRouter(app, m.Controller, m.BrowsingPane)
	// inject controller dependencies
	m.Controller.NavHandler = m.Router.NavigateTo
	m.Controller.ReloadFunc = m.BrowsingPane.Reload
	m.Controller.CurPageFunc = m.BrowsingPane.CurPage

	m.BottomPanel = NewBottomPanel(app.Player, m.Controller)
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
		m.Window.SetTitle(fmt.Sprintf("%s – %s · %s", song.Title, song.Artist, appName))
	})
	app.ServerManager.OnServerConnected(func() {
		m.BrowsingPane.EnableNavigationButtons()
		m.Router.NavigateTo(m.StartupPage())
		// check if found new version on startup
		if t := app.UpdateChecker.VersionTagFound(); t != "" && t != app.Config.Application.LastCheckedVersion {
			if t != app.VersionTag() {
				m.ShowNewVersionDialog(appName, t)
			}
			m.App.Config.Application.LastCheckedVersion = t
		}
		// register callback for the ongoing periodic update check
		m.App.UpdateChecker.OnUpdatedVersionFound = func() {
			t := m.App.UpdateChecker.VersionTagFound()
			if t != app.VersionTag() {
				m.ShowNewVersionDialog(appName, t)
			}
			m.App.Config.Application.LastCheckedVersion = t
		}
	})
	app.ServerManager.OnLogout(func() {
		m.BrowsingPane.DisableNavigationButtons()
		m.BrowsingPane.SetPage(nil)
		m.BrowsingPane.ClearHistory()
		m.Controller.PromptForLoginAndConnect()
	})
	m.BrowsingPane.AddSettingsMenuItem("Log Out", app.ServerManager.Logout)
	m.BrowsingPane.AddSettingsMenuItem("Check for Updates", func() {
		go func() {
			if t := app.UpdateChecker.CheckLatestVersionTag(); t != "" && t != app.VersionTag() {
				m.ShowNewVersionDialog(appName, t)
			} else {
				dialog.ShowInformation("No new version found",
					"You are running the latest version of "+appName,
					m.Window)
			}
		}()
	})
	m.BrowsingPane.AddSettingsMenuItem("Settings...", m.Controller.ShowSettingsDialog)
	m.BrowsingPane.AddSettingsMenuItem("About...", m.Controller.ShowAboutDialog)
	m.addNavigationButtons()
	m.BrowsingPane.DisableNavigationButtons()
	m.addShortcuts()
	return m
}

func (m *MainWindow) StartupPage() controller.Route {
	switch m.App.Config.Application.StartupPage {
	case "Favorites":
		return controller.FavoritesRoute()
	case "Playlists":
		return controller.PlaylistsRoute()
	default:
		return controller.AlbumsRoute()
	}
}

func (m *MainWindow) SetupSystemTrayMenu(appName string, fyneApp fyne.App) {
	if desk, ok := fyneApp.(desktop.App); ok {
		menu := fyne.NewMenu(appName,
			fyne.NewMenuItem("Play/Pause", func() {
				_ = m.App.Player.PlayPause()
			}),
			fyne.NewMenuItem("Previous", func() {
				_ = m.App.Player.SeekBackOrPrevious()
			}),
			fyne.NewMenuItem("Next", func() {
				_ = m.App.Player.SeekNext()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Show", m.Window.Show),
			fyne.NewMenuItem("Hide", m.Window.Hide),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(res.ResAppicon256Png)
		m.haveSystemTray = true
	}
}

func (m *MainWindow) HaveSystemTray() bool {
	return m.haveSystemTray
}

func (m *MainWindow) ShowNewVersionDialog(appName, versionTag string) {
	contentStr := fmt.Sprintf("A new version of %s (%s) is available",
		appName, versionTag)
	m.Controller.QueueShowModalFunc(func() {
		dialog.ShowCustomConfirm("A new version is available",
			"Go to release page", "Skip this version",
			widget.NewLabel(contentStr), func(show bool) {
				if show {
					fyne.CurrentApp().OpenURL(m.App.UpdateChecker.LatestReleaseURL())
				}
				m.App.Config.Application.LastCheckedVersion = versionTag
			}, m.Window)
	})
}

func (m *MainWindow) addNavigationButtons() {
	m.BrowsingPane.AddNavigationButton(res.ResHeadphonesInvertPng, func() {
		m.Router.NavigateTo(controller.NowPlayingRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResHeartFilledInvertPng, func() {
		m.Router.NavigateTo(controller.FavoritesRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResDiscInvertPng, func() {
		m.Router.NavigateTo(controller.AlbumsRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResPeopleInvertPng, func() {
		m.Router.NavigateTo(controller.ArtistsRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResTheatermasksInvertPng, func() {
		m.Router.NavigateTo(controller.GenresRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResPlaylistInvertPng, func() {
		m.Router.NavigateTo(controller.PlaylistsRoute())
	})
	m.BrowsingPane.AddNavigationButton(res.ResMusicnotesInvertPng, func() {
		m.Router.NavigateTo(controller.TracksRoute())
	})
}

func (m *MainWindow) addShortcuts() {
	for _, sh := range os.BackShortcuts {
		m.Canvas().AddShortcut(&sh, func(_ fyne.Shortcut) {
			m.BrowsingPane.GoBack()
			// TODO: reset focus only if something inside the page had focus
			m.Canvas().Focus(nil)
		})
	}
	for _, sh := range os.ForwardShortcuts {
		m.Canvas().AddShortcut(&sh, func(_ fyne.Shortcut) {
			m.BrowsingPane.GoForward()
			m.Canvas().Focus(nil)
		})
	}

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
	m.Canvas().AddShortcut(&ShortcutCloseWindow, func(_ fyne.Shortcut) {
		if m.App.Config.Application.CloseToSystemTray && m.HaveSystemTray() {
			m.Window.Hide()
		}
	})

	for i, ns := range NavShortcuts {
		m.Canvas().AddShortcut(&ns, func(i int) func(fyne.Shortcut) {
			return func(fyne.Shortcut) {
				m.BrowsingPane.ActivateNavigationButton(i)
			}
		}(i))
	}

	m.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		switch e.Name {
		case fyne.KeyEscape:
			m.Controller.CloseEscapablePopUp()
		case fyne.KeySpace:
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
