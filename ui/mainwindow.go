package ui

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"

	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui/browsing"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/dialogs"
	"github.com/dweymouth/supersonic/ui/shortcuts"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	Window fyne.Window

	App          *backend.App
	Router       browsing.Router
	Controller   *controller.Controller
	BrowsingPane *browsing.BrowsingPane
	BottomPanel  *BottomPanel
	ToastOverlay *ToastOverlay

	theme            *theme.MyTheme
	haveSystemTray   bool
	alreadyConnected bool // tracks if we have already connected to a server before
	content          *mainWindowContent

	// needs to bes shown/hidden when switching between servers based on whether they support radio
	radioBtn fyne.CanvasObject
}

func NewMainWindow(fyneApp fyne.App, appName, displayAppName, appVersion string, app *backend.App) MainWindow {
	m := MainWindow{
		App:    app,
		Window: fyneApp.NewWindow(displayAppName),
		theme:  theme.NewMyTheme(&app.Config.Theme, app.ThemesDir()),
	}
	fynetooltip.SetToolTipTextSizeName(theme.SizeNameSubText)

	m.theme.NormalFont = app.Config.Application.FontNormalTTF
	m.theme.BoldFont = app.Config.Application.FontBoldTTF
	fyneApp.Settings().SetTheme(m.theme)

	if app.Config.Application.EnableSystemTray {
		m.SetupSystemTrayMenu(displayAppName, fyneApp)
	}
	m.Controller = controller.New(app, appVersion, m.Window)
	m.BrowsingPane = browsing.NewBrowsingPane(app, m.Controller, func() { m.Router.NavigateTo(m.StartupPage()) })
	m.ToastOverlay = NewToastOverlay()
	m.Router = browsing.NewRouter(app, m.Controller, m.BrowsingPane)
	// inject controller dependencies
	m.Controller.NavHandler = m.Router.NavigateTo
	m.Controller.ReloadFunc = m.BrowsingPane.Reload
	m.Controller.CurPageFunc = m.BrowsingPane.CurrentPage
	m.Controller.RefreshPageFunc = m.BrowsingPane.RefreshPage
	m.Controller.SelectAllPageFunc = m.BrowsingPane.SelectAll
	m.Controller.UnselectAllPageFunc = m.BrowsingPane.UnselectAll
	m.Controller.ToastProvider = m.ToastOverlay

	if runtime.GOOS == "darwin" {
		// Fyne will extract out an "About" menu item and
		// assign its action to the native Mac "About Supersonic" menu item
		m.Window.SetMainMenu(fyne.NewMainMenu(
			fyne.NewMenu("File", /*name doesn't matter*/
				fyne.NewMenuItem("About", func() {
					m.Window.Show()
					m.Controller.ShowAboutDialog()
				})),
		))
	}

	m.BottomPanel = NewBottomPanel(app.PlaybackManager, app.ImageManager, m.Controller)
	app.PlaybackManager.OnSongChange(func(item mediaprovider.MediaItem, _ *mediaprovider.Track) {
		fyne.Do(func() { m.UpdateOnTrackChange(item) })
	})
	app.ServerManager.OnServerConnected(func() {
		go m.RunOnServerConnectedTasks(app, displayAppName)
	})
	app.ServerManager.OnLogout(func() {
		m.BrowsingPane.DisableNavigationButtons()
		m.BrowsingPane.SetPage(nil)
		m.BrowsingPane.ClearHistory()
		m.Controller.PromptForLoginAndConnect()
	})
	m.BrowsingPane.AddSettingsMenuItem(lang.L("Log Out"), func() { app.ServerManager.Logout(true) })
	m.BrowsingPane.AddSettingsMenuItem(lang.L("Switch Servers"), func() { app.ServerManager.Logout(false) })
	m.BrowsingPane.AddSettingsMenuItem(lang.L("Rescan Library"), func() { app.ServerManager.Server.RescanLibrary() })
	m.BrowsingPane.AddSettingsMenuSeparator()
	m.BrowsingPane.AddSettingsSubmenu(lang.L("Visualizations"),
		fyne.NewMenu("", []*fyne.MenuItem{
			fyne.NewMenuItem(lang.L("Peak Meter"), m.Controller.ShowPeakMeter),
		}...))
	m.BrowsingPane.AddSettingsMenuSeparator()
	m.BrowsingPane.AddSettingsMenuItem(lang.L("Check for Updates"), func() {
		go func() {
			if t := app.UpdateChecker.CheckLatestVersionTag(); t != "" && t != app.VersionTag() {
				m.ShowNewVersionDialog(displayAppName, t)
			} else {
				dialog.ShowInformation(lang.L("No new version found"),
					lang.L("You are running the latest version of")+" "+displayAppName,
					m.Window)
			}
		}()
	})
	m.BrowsingPane.AddSettingsMenuItem(lang.L("Settings")+"...", m.showSettingsDialog)
	m.BrowsingPane.AddSettingsMenuItem(lang.L("About")+"...", m.Controller.ShowAboutDialog)
	m.addNavigationButtons()
	m.BrowsingPane.DisableNavigationButtons()
	m.addShortcuts()

	center := container.NewStack(m.BrowsingPane, m.ToastOverlay)
	m.content = newMainWindowContent(container.NewBorder(nil, m.BottomPanel, nil, nil, center),
		m.Controller.UnselectAll)
	m.Window.SetContent(fynetooltip.AddWindowToolTipLayer(m.content, m.Window.Canvas()))
	m.setInitialSize()

	m.Window.SetCloseIntercept(func() {
		m.SaveWindowSize()
		if app.Config.Application.CloseToSystemTray && m.HaveSystemTray() {
			m.Window.Hide()
		} else {
			m.Window.Close()
		}
	})

	return m
}

func (m *MainWindow) UpdateOnTrackChange(item mediaprovider.MediaItem) {
	if item == nil {
		m.Window.SetTitle(res.DisplayName)
		return
	}
	meta := item.Metadata()
	artistDisp := ""
	if tr, ok := item.(*mediaprovider.Track); ok {
		artistDisp = " – " + strings.Join(tr.ArtistNames, ", ")
	}
	m.Window.SetTitle(fmt.Sprintf("%s%s · %s", meta.Name, artistDisp, res.DisplayName))
	if m.App.Config.Application.ShowTrackChangeNotification {
		if runtime.GOOS == "linux" {
			if notifySend, err := exec.LookPath("notify-send"); err == nil {
				go func() {
					args := []string{
						"--app-name", "supersonic",
						"--urgency", "low",
						"--expire-time", "10000",
						// replace previous notification
						"--hint", "string:x-canonical-private-synchronous:supersonic-track",
						meta.Name, strings.TrimPrefix(artistDisp, " – "),
					}

					app.ImageManager.GetCoverThumbnail(meta.CoverArtID)
					if path, err := app.ImageManager.GetCoverArtPath(meta.CoverArtID); err == nil {
						args = append([]string{"--icon", path}, args...)
					}

					if out, err := exec.Command(notifySend, args...).CombinedOutput(); err != nil {
						log.Printf("notify-send error: %s %s", strings.TrimSpace(string(out)), err)
					}
				}()
				return
			}
		}

		// TODO: Once Fyne issue #2935 is resolved, show album cover
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   meta.Name,
			Content: artistDisp,
		})
	}
}

func (m *MainWindow) DesiredSize() fyne.Size {
	w := float32(m.App.Config.Application.WindowWidth)
	if w <= 1 {
		w = 1000
	}
	h := float32(m.App.Config.Application.WindowHeight)
	if h <= 1 {
		h = 800
	}
	return fyne.NewSize(w, h)
}

func (m *MainWindow) setInitialSize() {
	m.Window.Resize(m.DesiredSize())
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

func (m *MainWindow) RunOnServerConnectedTasks(app *backend.App, displayAppName string) {
	time.Sleep(1 * time.Millisecond) // ensure this runs after sync tasks

	if app.Config.Application.SavePlayQueue {
		go func() {
			if err := app.LoadSavedPlayQueue(); err != nil {
				log.Printf("failed to load saved play queue: %s", err.Error())
			}
		}()
	}

	fyne.Do(func() {
		m.BrowsingPane.EnableNavigationButtons()
		m.Router.NavigateTo(m.StartupPage())
		_, canRate := m.App.ServerManager.Server.(mediaprovider.SupportsRating)
		m.BottomPanel.NowPlaying.DisableRating = !canRate

		_, supportsRadio := m.App.ServerManager.Server.(mediaprovider.RadioProvider)
		if supportsRadio {
			m.radioBtn.Show()
		} else {
			m.radioBtn.Hide()
		}
	})

	m.App.SaveConfigFile()

	if m.alreadyConnected {
		// below tasks only need to run on first time connecting to a server since launch
		return
	}

	// check if launching new version, else if found available update on startup
	fyne.Do(func() {
		if l := app.Config.Application.LastLaunchedVersion; app.VersionTag() != l {
			if !app.IsFirstLaunch() {
				m.ShowWhatsNewDialog()
			}
			m.App.Config.Application.LastLaunchedVersion = app.VersionTag()
		} else if t := app.UpdateChecker.VersionTagFound(); t != "" && t != app.Config.Application.LastCheckedVersion {
			if t != app.VersionTag() {
				m.ShowNewVersionDialog(displayAppName, t)
			}
			m.App.Config.Application.LastCheckedVersion = t
		}
	})

	// register callback for the ongoing periodic update check
	m.App.UpdateChecker.OnUpdatedVersionFound = func() {
		t := m.App.UpdateChecker.VersionTagFound()
		if t != app.VersionTag() {
			m.ShowNewVersionDialog(displayAppName, t)
		}
		m.App.Config.Application.LastCheckedVersion = t
	}
	m.alreadyConnected = true
}

func (m *MainWindow) SetupSystemTrayMenu(appName string, fyneApp fyne.App) {
	if desk, ok := fyneApp.(desktop.App); ok {
		menu := fyne.NewMenu(appName,
			fyne.NewMenuItem(fmt.Sprintf("%s/%s", lang.L("Play"), lang.L("Pause")), func() {
				m.App.PlaybackManager.PlayPause()
			}),
			fyne.NewMenuItem(lang.L("Previous"), func() {
				m.App.PlaybackManager.SeekBackOrPrevious()
			}),
			fyne.NewMenuItem(lang.L("Next"), func() {
				m.App.PlaybackManager.SeekNext()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(lang.L("Volume")+" +10%", func() {
				vol := m.App.PlaybackManager.Volume()
				vol = vol + int(float64(vol)*0.1)
				// will clamp to range for us
				m.App.PlaybackManager.SetVolume(vol)
			}),
			fyne.NewMenuItem(lang.L("Volume")+" -10%", func() {
				vol := m.App.PlaybackManager.Volume()
				vol = vol - int(float64(vol)*0.1)
				m.App.PlaybackManager.SetVolume(vol)
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(lang.L("Show"), func() { fyne.Do(m.Window.Show) }),
			fyne.NewMenuItem(lang.L("Hide"), func() { fyne.Do(m.Window.Hide) }),
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
		dialog.ShowCustomConfirm(lang.L("A new version is available"),
			lang.L("Go to release page"), lang.L("Skip this version"),
			widget.NewLabel(contentStr), func(show bool) {
				if show {
					fyne.CurrentApp().OpenURL(m.App.UpdateChecker.LatestReleaseURL())
				}
				m.App.Config.Application.LastCheckedVersion = versionTag
			}, m.Window)
	})
}

func (m *MainWindow) ShowWhatsNewDialog() {
	dialog.ShowCustom("What's new in "+res.AppVersion, lang.L("Close"), dialogs.NewWhatsNewDialog(), m.Window)
}

func (m *MainWindow) addNavigationButtons() {
	m.BrowsingPane.AddNavigationButton(theme.NowPlayingIcon, controller.NowPlaying, func() {
		m.Router.NavigateTo(controller.NowPlayingRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.FavoriteIcon, controller.Favorites, func() {
		m.Router.NavigateTo(controller.FavoritesRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.AlbumIcon, controller.Albums, func() {
		m.Router.NavigateTo(controller.AlbumsRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.ArtistIcon, controller.Artists, func() {
		m.Router.NavigateTo(controller.ArtistsRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.GenreIcon, controller.Genres, func() {
		m.Router.NavigateTo(controller.GenresRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.PlaylistIcon, controller.Playlists, func() {
		m.Router.NavigateTo(controller.PlaylistsRoute())
	})
	m.BrowsingPane.AddNavigationButton(theme.TracksIcon, controller.Tracks, func() {
		m.Router.NavigateTo(controller.TracksRoute())
	})
	m.radioBtn = m.BrowsingPane.AddNavigationButton(theme.RadioIcon, controller.Radios, func() {
		m.Router.NavigateTo(controller.RadiosRoute())
	})
}

func (m *MainWindow) addShortcuts() {
	for _, sh := range shortcuts.BackShortcuts {
		m.Canvas().AddShortcut(&sh, func(_ fyne.Shortcut) {
			m.BrowsingPane.GoBack()
		})
	}
	for _, sh := range shortcuts.ForwardShortcuts {
		m.Canvas().AddShortcut(&sh, func(_ fyne.Shortcut) {
			m.BrowsingPane.GoForward()
		})
	}
	if shortcuts.SettingsShortcut != nil {
		m.Canvas().AddShortcut(shortcuts.SettingsShortcut, func(_ fyne.Shortcut) {
			m.showSettingsDialog()
		})
	}
	if shortcuts.QuitShortcut != nil {
		m.Canvas().AddShortcut(shortcuts.QuitShortcut, func(_ fyne.Shortcut) {
			m.Quit()
		})
	}

	m.Canvas().AddShortcut(&shortcuts.ShortcutReload, func(_ fyne.Shortcut) {
		m.BrowsingPane.Reload()
	})
	m.Canvas().AddShortcut(&shortcuts.ShortcutSearch, func(_ fyne.Shortcut) {
		if m.Controller.HaveModal() {
			// Do not focus search widget behind modal dialog
			return
		}
		if s := m.BrowsingPane.GetSearchBarIfAny(); s != nil {
			m.Window.Canvas().Focus(s)
		}
	})
	m.Canvas().AddShortcut(&shortcuts.ShortcutQuickSearch, func(_ fyne.Shortcut) {
		if !m.Controller.HaveModal() {
			m.Controller.ShowQuickSearch()
		}
	})
	m.Canvas().AddShortcut(&fyne.ShortcutSelectAll{}, func(_ fyne.Shortcut) {
		m.Controller.SelectAll()
	})
	m.Canvas().AddShortcut(&shortcuts.ShortcutCloseWindow, func(_ fyne.Shortcut) {
		if m.App.Config.Application.CloseToSystemTray && m.HaveSystemTray() {
			m.Window.Hide()
		}
	})

	for i, ns := range shortcuts.NavShortcuts {
		m.Canvas().AddShortcut(&ns, func(i int) func(fyne.Shortcut) {
			return func(fyne.Shortcut) {
				m.BrowsingPane.ActivateNavigationButton(i)
			}
		}(i))
	}

	m.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		switch e.Name {
		case fyne.KeyUp:
			m.BrowsingPane.ScrollUp()
		case fyne.KeyDown:
			m.BrowsingPane.ScrollDown()
		case fyne.KeyPageUp:
			m.BrowsingPane.PageUp()
		case fyne.KeyPageDown:
			m.BrowsingPane.PageDown()
		case fyne.KeyEscape:
			m.Controller.CloseEscapablePopUp()
		case fyne.KeySpace:
			m.App.PlaybackManager.PlayPause()
		}
	})
	m.Canvas().SetOnMouseBack(m.BrowsingPane.GoBack)
	m.Canvas().SetOnMouseForward(m.BrowsingPane.GoForward)
}

func (m *MainWindow) showSettingsDialog() {
	m.Controller.ShowSettingsDialog(func() {
		fyne.CurrentApp().Settings().SetTheme(m.theme)
	}, m.theme.ListThemeFiles())
}

func (m *MainWindow) Show() {
	m.Window.Show()
}

func (m *MainWindow) ShowAndRun() {
	m.Window.ShowAndRun()
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

func (m *MainWindow) Quit() {
	m.SaveWindowSize()
	fyne.CurrentApp().Quit()
}

func (m *MainWindow) SaveWindowSize() {
	util.SaveWindowSize(m.Window,
		&m.App.Config.Application.WindowWidth,
		&m.App.Config.Application.WindowHeight)
}

// widget just so we can catch a tap event that doesn't land anywhere else
// and call BrowsingPane.UnselectAll()
type mainWindowContent struct {
	widget.BaseWidget

	content  fyne.CanvasObject
	onTapped func()
}

func newMainWindowContent(content fyne.CanvasObject, onTapped func()) *mainWindowContent {
	w := &mainWindowContent{content: content, onTapped: onTapped}
	w.ExtendBaseWidget(w)
	return w
}

func (m *mainWindowContent) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.content)
}

func (m *mainWindowContent) Tapped(*fyne.PointEvent) {
	m.onTapped()
}
