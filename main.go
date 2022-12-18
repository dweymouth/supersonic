package main

import (
	"context"
	"fmt"
	"gomuse/backend"
	"gomuse/player"
	"gomuse/ui"
	"log"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/20after4/configdir"
	"github.com/dweymouth/go-subsonic"
)

const appname = "gomuse"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cachePath := configdir.LocalCache(appname)
	log.Println(cachePath)

	p := player.NewWithClientName(appname)
	fmt.Println(p.Init())

	s := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    "***REMOVED***",
		User:       "drew",
		ClientName: appname,
	}

	lm := backend.NewLibraryManager(s)
	pm := backend.NewPlaybackManager(ctx, s, p)
	im := backend.NewImageManager(s, configdir.LocalCache(appname, "covers"))

	if err := s.Authenticate("***REMOVED***"); err != nil {
		fmt.Printf("error authenticating: %v\n", err)
	}

	myApp := app.New()
	myWindow := myApp.NewWindow(appname)

	b := ui.NewBottomPanel(p, pm, im)
	ag := ui.NewAlbumGrid(lm.RecentlyAddedIter(), pm, im)
	c := container.NewBorder(nil, b, nil, nil, ag)

	pm.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			myWindow.SetTitle("gomuse")
			return
		}
		myWindow.SetTitle(song.Title)
	})

	myWindow.SetContent(c)
	myWindow.Resize(fyne.NewSize(1350, 1000))
	myWindow.Show()

	myApp.Run()
	cancel()
	p.Destroy()
}
