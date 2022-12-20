package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"supersonic/backend"
	"supersonic/player"
	"supersonic/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/20after4/configdir"
	"github.com/dweymouth/go-subsonic"
)

const (
	appname    = "supersonic"
	configFile = "config.toml"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	p := player.NewWithClientName(appname)
	if err := p.Init(); err != nil {
		log.Fatalf("failed to initialize mpv player: %s", err.Error())
	}

	myApp := app.New()
	myWindow := myApp.NewWindow(appname)

	b := ui.NewBottomPanel(p)
	c := container.NewBorder(nil, b, nil, nil, &layout.Spacer{})

	myWindow.SetContent(c)
	myWindow.Resize(fyne.NewSize(1000, 800))
	myWindow.Show()

	s := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    "***REMOVED***",
		User:       "drew",
		ClientName: appname,
	}

	if err := s.Authenticate("***REMOVED***"); err != nil {
		fmt.Printf("error authenticating: %v\n", err)
	}

	lm := backend.NewLibraryManager(s)
	pm := backend.NewPlaybackManager(ctx, s, p)
	im := backend.NewImageManager(s, configdir.LocalCache(appname, "covers"))
	b.ImageManager = im

	b.SetPlaybackManager(pm)
	pm.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			myWindow.SetTitle("gomuse")
			return
		}
		myWindow.SetTitle(song.Title)
	})

	ag := ui.NewAlbumGrid(lm.RecentlyAddedIter(), pm, im)
	myWindow.SetContent(container.NewBorder(nil, b, nil, nil, ag))
	myApp.Run()
	cancel()
	p.Destroy()
}
