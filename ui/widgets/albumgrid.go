package widgets

import (
	"context"
	"image"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type ImageFetcher interface {
	GetAlbumThumbnailFromCache(string) (image.Image, bool)
	GetAlbumThumbnail(string) (image.Image, error)
}

type AlbumIterator interface {
	NextN(int, func(*subsonic.AlbumID3))
}

type AlbumGrid struct {
	widget.BaseWidget

	AlbumGridState

	grid *widget.GridWrapList
}

type AlbumGridState struct {
	albums   []*subsonic.AlbumID3
	iter     AlbumIterator
	fetching bool
	done     bool
	showYear bool

	imageFetcher     ImageFetcher
	OnPlayAlbum      func(string)
	OnShowAlbumPage  func(string)
	OnShowArtistPage func(string)

	scrollPos float32
}

var _ fyne.Widget = (*AlbumGrid)(nil)

func NewFixedAlbumGrid(albums []*subsonic.AlbumID3, fetch ImageFetcher, showYear bool) *AlbumGrid {
	ag := &AlbumGrid{
		AlbumGridState: AlbumGridState{
			albums:       albums,
			done:         true,
			imageFetcher: fetch,
			showYear:     showYear,
		},
	}
	ag.ExtendBaseWidget(ag)
	ag.createGridWrapList()
	return ag
}

func NewAlbumGrid(iter AlbumIterator, fetch ImageFetcher, showYear bool) *AlbumGrid {
	ag := &AlbumGrid{
		AlbumGridState: AlbumGridState{
			iter:         iter,
			imageFetcher: fetch,
		},
	}
	ag.ExtendBaseWidget(ag)

	ag.createGridWrapList()

	// fetch initial albums
	ag.fetchMoreAlbums(36)
	return ag
}

func (ag *AlbumGrid) SaveToState() AlbumGridState {
	s := ag.AlbumGridState
	s.scrollPos = ag.grid.GetScrollOffset()
	return s
}

func NewAlbumGridFromState(state AlbumGridState) *AlbumGrid {
	ag := &AlbumGrid{AlbumGridState: state}
	ag.ExtendBaseWidget(ag)
	ag.createGridWrapList()
	ag.Refresh() // needed to initialize the widget
	ag.grid.ScrollToOffset(state.scrollPos)
	return ag
}

func (ag *AlbumGrid) Clear() {
	ag.albums = nil
	ag.done = true
}

func (ag *AlbumGrid) Reset(iter AlbumIterator) {
	ag.albums = nil
	ag.fetching = false
	ag.done = false
	ag.iter = iter
	ag.fetchMoreAlbums(36)
}

func (ag *AlbumGrid) createGridWrapList() {
	g := widget.NewGridWrapList(
		func() int {
			return len(ag.albums)
		},
		// create func
		func() fyne.CanvasObject {
			ac := NewAlbumCard(ag.showYear)
			ac.OnPlay = func() {
				if ag.OnPlayAlbum != nil {
					ag.OnPlayAlbum(ac.AlbumID())
				}
			}
			ac.OnShowArtistPage = func() {
				if ag.OnShowArtistPage != nil {
					ag.OnShowArtistPage(ac.ArtistID())
				}
			}
			ac.OnShowAlbumPage = func() {
				if ag.OnShowAlbumPage != nil {
					ag.OnShowAlbumPage(ac.AlbumID())
				}
			}
			return ac
		},
		// update func
		func(itemID int, obj fyne.CanvasObject) {
			ac := obj.(*AlbumCard)
			ag.doUpdateAlbumCard(itemID, ac)
		},
	)
	ag.grid = g
}

func (ag *AlbumGrid) doUpdateAlbumCard(albumIdx int, ac *AlbumCard) {
	album := ag.albums[albumIdx]
	if ac.PrevAlbumID == album.ID {
		// nothing to do
		return
	}
	ac.Update(album)
	ac.PrevAlbumID = album.ID
	// TODO: set image to a placeholder before spinning off async fetch
	// cancel any previous image fetch
	if ac.ImgLoadCancel != nil {
		ac.ImgLoadCancel()
		ac.ImgLoadCancel = nil
	}
	if img, ok := ag.imageFetcher.GetAlbumThumbnailFromCache(album.ID); ok {
		ac.Cover.SetImage(img)
		ac.Cover.Refresh()
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		go func(ctx context.Context) {
			i, err := ag.imageFetcher.GetAlbumThumbnail(album.ID)
			select {
			case <-ctx.Done():
				return
			default:
				if err == nil {
					ac.Cover.SetImage(i)
					ac.Cover.Refresh()
				} else {
					log.Printf("error fetching image: %s", err.Error())
				}
			}
		}(ctx)
		ac.ImgLoadCancel = cancel
	}

	// TODO: remove magic number 10
	if !ag.done && !ag.fetching && albumIdx > len(ag.albums)-10 {
		ag.fetchMoreAlbums(20)
	}
}

func (a *AlbumGrid) fetchMoreAlbums(count int) {
	if a.iter == nil {
		a.done = true
	}
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
		a.Refresh()
	})
}

func (a *AlbumGrid) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.grid)
}
