package main

import (
	"log"
	"math"
	"runtime"
	"time"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	appname          = "supersonic"
	displayName      = "Supersonic"
	appVersion       = "0.5.2"
	appVersionTag    = "v" + appVersion
	configFile       = "config.toml"
	latestReleaseURL = "https://github.com/dweymouth/supersonic/releases/latest"
)

func main() {
	myApp, err := backend.StartupApp(appname, displayName, appVersionTag, configFile, latestReleaseURL)
	if err != nil {
		log.Fatalf("fatal startup error: %v", err.Error())
	}

	fyneApp := app.New()
	fyneApp.SetIcon(res.ResAppicon256Png)

	w := float32(myApp.Config.Application.WindowWidth)
	if w <= 1 {
		w = 1000
	}
	h := float32(myApp.Config.Application.WindowHeight)
	if h <= 1 {
		h = 800
	}
	mainWindow := ui.NewMainWindow(fyneApp, appname, displayName, appVersion, myApp, fyne.NewSize(w, h))
	myApp.OnReactivate = mainWindow.Show
	myApp.OnExit = func() {
		saveWindowSize(myApp.Config, mainWindow.Window)
		fyneApp.Quit()
	}

	go func() {
		// TODO: There is a race condition with laying out the window before the
		// window creation animation on Ubuntu (and other DEs?) finishes, where
		// the window will be misdrawn into a smaller area if the animation hasn't finished.
		// This makes it much less likely to occur (not seen on dozens of startups)
		// but is a hacky "solution"!
		if runtime.GOOS == "linux" {
			time.Sleep(250 * time.Millisecond)
		}
		defaultServer := myApp.ServerManager.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.Controller.PromptForFirstServer()
		} else {
			mainWindow.Controller.DoConnectToServerWorkflow(defaultServer)
		}
	}()

	mainWindow.Show()
	mainWindow.Window.SetCloseIntercept(func() {
		saveWindowSize(myApp.Config, mainWindow.Window)
		if myApp.Config.Application.CloseToSystemTray &&
			mainWindow.HaveSystemTray() {
			mainWindow.Window.Hide()
		} else {
			fyneApp.Quit()
		}
	})
	fyneApp.Run()

	log.Println("Running shutdown tasks...")
	myApp.Shutdown()
}

func saveWindowSize(config *backend.Config, window fyne.Window) {
	// round sizes to even to avoid Wayland issues with 2x scaling factor
	// https://github.com/dweymouth/supersonic/issues/212
	config.Application.WindowHeight = int(math.RoundToEven(float64(window.Canvas().Size().Height)))
	config.Application.WindowWidth = int(math.RoundToEven(float64(window.Canvas().Size().Width)))
}
