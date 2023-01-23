package main

import (
	"log"
	"path"
	"supersonic/backend"
	"supersonic/ui"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/20after4/configdir"
	"github.com/zalando/go-keyring"
)

const (
	appname    = "supersonic"
	configFile = "config.toml"
)

func configPath() string {
	return path.Join(configdir.LocalConfig(appname), configFile)
}

func main() {
	myApp, err := backend.StartupApp()
	if err != nil {
		log.Fatalf("fatal startup error: %v", err.Error())
	}

	fyneApp := app.New()
	fyneApp.Settings().SetTheme(&ui.MyTheme{})
	w := float32(myApp.Config.Application.WindowWidth)
	if w <= 1 {
		w = 1000
	}
	h := float32(myApp.Config.Application.WindowHeight)
	if h <= 1 {
		h = 800
	}
	mainWindow := ui.NewMainWindow(fyneApp, appname, myApp, fyne.NewSize(w, h))

	// TODO: There is some race condition with running this initial startup
	// task immediately before showing and running the window/Fyne main loop where
	// the window can occasionally get misdrawn on startup. (Only seen on Ubuntu so far).
	// This makes it much less likely to occur (not seen on dozens of startups)
	// but is a hacky "solution"!
	go func() {
		time.Sleep(250 * time.Millisecond)
		defaultServer := myApp.Config.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.PromptForFirstServer(func(nick, host, user, pass string) {
				server := myApp.Config.AddServer(nick, host, user)
				err := keyring.Set(appname, server.ID.String(), pass)
				if err != nil {
					log.Printf("error setting keyring credentials: %v", err)
					// TODO: handle?
				}
				setupServer(myApp, server)
			})
		} else {
			setupServer(myApp, defaultServer)
		}
	}()

	mainWindow.Show()
	mainWindow.Window.SetCloseIntercept(func() {
		myApp.Config.Application.WindowHeight = int(mainWindow.Canvas().Size().Height)
		myApp.Config.Application.WindowWidth = int(mainWindow.Canvas().Size().Width)
		mainWindow.Window.Close()
	})
	fyneApp.Run()
	myApp.Config.WriteConfigFile(configPath())
	myApp.Shutdown()

}

func setupServer(app *backend.App, server *backend.ServerConfig) {
	pass, err := keyring.Get(appname, server.ID.String())
	if err != nil {
		log.Printf("error getting password from keyring: %v", err)
	}
	if err := app.ServerManager.ConnectToServer(server, pass); err != nil {
		log.Printf("error connecting to server: %v", err)
		// TODO: surface error to user
	}
}
