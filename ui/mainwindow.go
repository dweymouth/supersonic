package ui

import (
	"supersonic/backend"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type MainWindow struct {
	Window fyne.Window

	BottomPanel *BottomPanel
}

func NewMainWindow(fyneApp fyne.App, appName string, app *backend.App) MainWindow {
	m := MainWindow{
		Window:      fyneApp.NewWindow(appName),
		BottomPanel: NewBottomPanel(app.Player),
	}
	m.BottomPanel.SetPlaybackManager(app.PlaybackManager)
	m.BottomPanel.ImageManager = app.ImageManager
	c := container.NewBorder(nil, m.BottomPanel, nil, nil, &layout.Spacer{})
	m.Window.SetContent(c)
	m.Window.Resize(fyne.NewSize(1000, 800))
	app.PlaybackManager.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			m.Window.SetTitle(appName)
			return
		}
		m.Window.SetTitle(song.Title)
	})
	app.ServerManager.OnServerConnected(func() {
		ag := NewAlbumGrid(app.LibraryManager.RecentlyAddedIter(), app.ImageManager.GetAlbumThumbnail)
		ag.OnPlayAlbum = func(albumID string) {
			_ = app.PlaybackManager.PlayAlbum(albumID)
		}
		m.Window.SetContent(container.NewBorder(nil, m.BottomPanel, nil, nil, ag))
	})
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
