package ui

import (
	"gomuse/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type AlbumGrid struct {
	widget.BaseWidget

	grid     *widget.GridWrapList
	albums   []*subsonic.AlbumID3
	iter     backend.AlbumIterator
	fetching bool
	done     bool
}

var _ fyne.Widget = (*AlbumGrid)(nil)

func NewAlbumGrid(iter backend.AlbumIterator, pm *backend.PlaybackManager, im *backend.ImageManager) *AlbumGrid {
	ag := &AlbumGrid{
		iter: iter,
	}
	ag.ExtendBaseWidget(ag)

	g := widget.NewGridWrapList(
		func() int {
			return len(ag.albums)
		},
		// create func
		func() fyne.CanvasObject {
			ac := NewAlbumCard()
			ac.OnTapped = func() {
				pm.PlayAlbum(ac.AlbumID())
			}
			return ac
		},
		// update func
		func(itemID int, obj fyne.CanvasObject) {
			ac := obj.(*AlbumCard)
			ac.Update(ag.albums[itemID])
			// TODO: set image to a placeholder before spinning off async fetch
			go func() {
				i, err := im.GetAlbumThumbnail(ag.albums[itemID].ID)
				if err == nil {
					ac.Cover.SetImage(i)
					ac.Refresh()
				}
			}()

			// TODO: remove magic number 10
			if !ag.done && !ag.fetching && itemID > len(ag.albums)-10 {
				ag.fetchMoreAlbums(10)
			}
		},
	)
	ag.grid = g

	// fetch initial albums
	ag.fetchMoreAlbums(36)
	return ag
}

func (a *AlbumGrid) fetchMoreAlbums(count int) {
	i := 0
	a.fetching = true
	a.iter.NextN(count, func(al *subsonic.AlbumID3) {
		if al == nil {
			a.done = true
			return
		}
		a.albums = append(a.albums, al)
		i++
		if i == count {
			a.fetching = false
		}
	})
}

func (a *AlbumGrid) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.grid)
}
