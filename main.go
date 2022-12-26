package main

import (
	"log"
	"path"
	"supersonic/backend"
	"supersonic/ui"

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
	mainWindow := ui.NewMainWindow(fyneApp, appname, myApp)
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

	mainWindow.Show()
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
