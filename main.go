package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	myApp, err := backend.StartupApp(res.AppName, res.DisplayName, res.AppVersionTag, res.LatestReleaseURL)
	if err != nil {
		log.Fatalf("fatal startup error: %v", err.Error())
	}

	if myApp.Config.Application.UIScaleSize == "Smaller" {
		os.Setenv("FYNE_SCALE", "0.85")
	} else if myApp.Config.Application.UIScaleSize == "Larger" {
		os.Setenv("FYNE_SCALE", "1.1")
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
	mainWindow := ui.NewMainWindow(fyneApp, res.AppName, res.DisplayName, res.AppVersion, myApp, fyne.NewSize(w, h))
	myApp.OnReactivate = mainWindow.Show
	myApp.OnExit = mainWindow.Quit

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
		mainWindow.SaveWindowSize()
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
