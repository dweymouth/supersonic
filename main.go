package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"supersonic/backend"
	"supersonic/player"
	"supersonic/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/20after4/configdir"
	"github.com/dweymouth/go-subsonic"
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
	// TODO: organize this whole file better. Move some stuff to mainwindow?

	ctx, cancel := context.WithCancel(context.Background())

	p := player.NewWithClientName(appname)
	if err := p.Init(); err != nil {
		log.Fatalf("failed to initialize mpv player: %s", err.Error())
	}

	configdir.MakePath(configdir.LocalConfig(appname))

	cfg, err := backend.ReadConfigFile(configPath())
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = backend.DefaultConfig()
	}
	server := cfg.GetDefaultServer()

	myApp := app.New()
	mainWindow := ui.NewMainWindow(myApp, appname, p)

	if server == nil {
		d := ui.NewAddServerDialog("Connect to Server")
		pop := widget.NewModalPopUp(d, mainWindow.Canvas())
		d.OnSubmit = func() {
			pop.Hide()
			server = cfg.AddServer(d.Nickname, d.Host, d.Username)
			err := keyring.Set(appname, server.ID.String(), d.Password)
			if err != nil {
				log.Printf("error setting keyring credentials: %v", err)
			}
			setupServer(ctx, mainWindow, p, server)
		}
		pop.Show()
	} else {
		setupServer(ctx, mainWindow, p, server)
	}

	mainWindow.Show()
	myApp.Run()
	cfg.WriteConfigFile(configPath())

	cancel()
	p.Destroy()
}

func setupServer(ctx context.Context, myWindow ui.MainWindow, p *player.Player, server *backend.ServerConfig) {
	pass, err := keyring.Get(appname, server.ID.String())
	if err != nil {
		log.Printf("error reading keyring credentials: %v", err)
	}

	s := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    server.Hostname,
		User:       server.Username,
		ClientName: appname,
	}

	if err := s.Authenticate(pass); err != nil {
		// TODO: error dialog
		fmt.Printf("error authenticating: %v\n", err)
	}

	lm := backend.NewLibraryManager(s)
	pm := backend.NewPlaybackManager(ctx, s, p)
	im := backend.NewImageManager(s, configdir.LocalCache(appname, "covers"))
	myWindow.BottomPanel.ImageManager = im

	myWindow.BottomPanel.SetPlaybackManager(pm)
	pm.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			myWindow.SetTitle("gomuse")
			return
		}
		myWindow.SetTitle(song.Title)
	})

	ag := ui.NewAlbumGrid(lm.RecentlyAddedIter(), pm, im)
	myWindow.SetContent(container.NewBorder(nil, myWindow.BottomPanel, nil, nil, ag))

}
