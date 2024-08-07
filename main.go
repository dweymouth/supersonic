package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
)

func main() {
	// parse cmd line flags - see backend/cmdlineoptions.go
	flag.Parse()
	if *backend.FlagVersion {
		fmt.Println(res.AppVersion)
		return
	}
	if *backend.FlagHelp {
		flag.Usage()
		return
	}
	// rest of flag actions are handled in backend.StartupApp

	myApp, err := backend.StartupApp(res.AppName, res.DisplayName, res.AppVersionTag, res.LatestReleaseURL)
	if err != nil {
		if err != backend.ErrAnotherInstance {
			log.Fatalf("fatal startup error: %v", err.Error())
		}
		return
	}

	if myApp.Config.Application.UIScaleSize == "Smaller" {
		os.Setenv("FYNE_SCALE", "0.85")
	} else if myApp.Config.Application.UIScaleSize == "Larger" {
		os.Setenv("FYNE_SCALE", "1.1")
	}

	lang.AddTranslationsFS(res.Translations, "translations")
	fyneApp := app.New()
	fyneApp.SetIcon(res.ResAppicon256Png)

	mainWindow := ui.NewMainWindow(fyneApp, res.AppName, res.DisplayName, res.AppVersion, myApp)
	myApp.OnReactivate = mainWindow.Show
	myApp.OnExit = mainWindow.Quit

	go func() {
		defaultServer := myApp.ServerManager.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.Controller.PromptForFirstServer()
		} else {
			mainWindow.Controller.DoConnectToServerWorkflow(defaultServer)
		}
	}()

	// slightly hacky workaround for https://github.com/fyne-io/fyne/issues/4964
	workaroundWindowSize := sync.OnceFunc(func() {
		s := mainWindow.DesiredSize()
		scale := mainWindow.Window.Canvas().Scale()
		s.Width *= scale
		s.Height *= scale
		// exported in Supersonic Fyne fork
		mainWindow.Window.ProcessResized(int(s.Width), int(s.Height))
	})
	fyneApp.Lifecycle().SetOnEnteredForeground(func() {
		workaroundWindowSize()
	})
	mainWindow.ShowAndRun()

	log.Println("Running shutdown tasks...")
	myApp.Shutdown()
}
