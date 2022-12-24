package ui

import (
	"supersonic/player"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

type MainWindow struct {
	Window fyne.Window

	BottomPanel *BottomPanel
}

func NewMainWindow(app fyne.App, appName string, p *player.Player) MainWindow {
	m := MainWindow{
		Window:      app.NewWindow(appName),
		BottomPanel: NewBottomPanel(p),
	}
	c := container.NewBorder(nil, m.BottomPanel, nil, nil, &layout.Spacer{})
	m.Window.SetContent(c)
	m.Window.Resize(fyne.NewSize(1000, 800))
	return m
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
