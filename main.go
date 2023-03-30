package main

import (
	"log"
	"supersonic/backend"
	"supersonic/ui"
	"supersonic/ui/theme"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	appname          = "supersonic"
	displayName      = "Supersonic"
	appVersion       = "0.0.1-alpha2"
	appVersionTag    = "v" + appVersion
	configFile       = "config.toml"
	latestReleaseURL = "https://github.com/dweymouth/supersonic/releases/latest"
)

func main() {
	myApp, err := backend.StartupApp(appname, appVersionTag, configFile, latestReleaseURL)
	if err != nil {
		log.Fatalf("fatal startup error: %v", err.Error())
	}

	fyneApp := app.New()
	fyneApp.Settings().SetTheme(&theme.MyTheme{})
	w := float32(myApp.Config.Application.WindowWidth)
	if w <= 1 {
		w = 1000
	}
	h := float32(myApp.Config.Application.WindowHeight)
	if h <= 1 {
		h = 800
	}
	mainWindow := ui.NewMainWindow(fyneApp, displayName, appVersion, myApp, fyne.NewSize(w, h))

	// TODO: There is some race condition with running this initial startup
	// task immediately before showing and running the window/Fyne main loop where
	// the window can occasionally get misdrawn on startup. (Only seen on Ubuntu so far).
	// This makes it much less likely to occur (not seen on dozens of startups)
	// but is a hacky "solution"!
	go func() {
		time.Sleep(250 * time.Millisecond)
		defaultServer := myApp.Config.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.Controller.PromptForFirstServer()
		} else {
			mainWindow.Controller.DoConnectToServerWorkflow(defaultServer)
		}
	}()

	mainWindow.Show()
	mainWindow.Window.SetCloseIntercept(func() {
		myApp.Config.Application.WindowHeight = int(mainWindow.Canvas().Size().Height)
		myApp.Config.Application.WindowWidth = int(mainWindow.Canvas().Size().Width)
		mainWindow.Window.Close()
	})
	fyneApp.Run()

	// shutdown tasks
	myApp.Shutdown()
}
