package main

import (
	"context"
	"fmt"
	"gomuse/backend"
	"gomuse/player"
	"gomuse/ui"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/dweymouth/go-subsonic"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	myApp := app.New()
	myWindow := myApp.NewWindow("gomuse")

	p := player.NewWithClientName("gomuse")
	fmt.Println(p.Init())

	s := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    "***REMOVED***",
		User:       "drew",
		ClientName: "gomuse",
	}

	lm := backend.NewLibraryManager(s)
	pm := backend.NewPlaybackManager(ctx, s, p)
	im := backend.NewImageManager(s)

	if err := s.Authenticate("***REMOVED***"); err != nil {
		fmt.Printf("error authenticating: %v\n", err)
	}

	/*
		if len(os.Args) > 1 {
			// search albums by args[1] and load first matching album
			log.Printf("Searching for %q\n", os.Args[1])
			if res, err := s.Search3(os.Args[1], map[string]string{}); err == nil {
				if len(res.Album) > 0 {
					log.Println("Got album search result")
					album := res.Album[0]
					pm.LoadAlbum(album.ID)
				}
			}
		}
	*/

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
