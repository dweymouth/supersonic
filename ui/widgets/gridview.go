package widgets

import (
	"context"
	"image"
	"log"
	"supersonic/backend"
	"supersonic/res"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

const albumFetchBatchSize = 6

type ImageFetcher interface {
	GetCoverThumbnailFromCache(string) (image.Image, bool)
	GetCoverThumbnail(string) (image.Image, error)
}

type GridView struct {
	widget.BaseWidget

	GridViewState

	grid *widget.GridWrapList
}

type GridViewState struct {
	items        []*subsonic.AlbumID3
	itemsMutex   sync.RWMutex
	iter         *backend.BatchingIterator
	highestShown int
	fetching     bool
	done         bool
	showYear     bool

	imageFetcher        ImageFetcher
	OnPlay              func(string)
	OnShowItemPage      func(string)
	OnShowSecondaryPage func(string)

	scrollPos float32
}

var _ fyne.Widget = (*GridView)(nil)

func NewFixedAlbumGrid(albums []*subsonic.AlbumID3, fetch ImageFetcher, showYear bool) *GridView {
	g := &GridView{
		GridViewState: GridViewState{
			items:        albums,
			done:         true,
			imageFetcher: fetch,
			showYear:     showYear,
		},
	}
	g.ExtendBaseWidget(g)
	g.createGridWrapList()
	return g
}

func NewAlbumGrid(iter backend.AlbumIterator, fetch ImageFetcher, showYear bool) *GridView {
	g := &GridView{
		GridViewState: GridViewState{
			iter:         backend.NewBatchingIterator(iter),
			imageFetcher: fetch,
		},
	}
	g.ExtendBaseWidget(g)

	g.createGridWrapList()

	// fetch initial items
	g.fetchMoreItems(36)
	return g
}

func (g *GridView) SaveToState() GridViewState {
	s := g.GridViewState
	s.scrollPos = g.grid.GetScrollOffset()
	return s
}

func NewGridViewFromState(state GridViewState) *GridView {
	g := &GridView{GridViewState: state}
	g.ExtendBaseWidget(g)
	g.createGridWrapList()
	g.Refresh() // needed to initialize the widget
	g.grid.ScrollToOffset(state.scrollPos)
	return g
}

func (g *GridView) Clear() {
	g.itemsMutex.Lock()
	defer g.itemsMutex.Unlock()
	g.items = nil
	g.done = true
}

func (g *GridView) Reset(iter backend.AlbumIterator) {
	g.itemsMutex.Lock()
	g.items = nil
	g.itemsMutex.Unlock()
	g.fetching = false
	g.done = false
	g.highestShown = 0
	g.iter = backend.NewBatchingIterator(iter)
	g.fetchMoreItems(36)
}

func (g *GridView) createGridWrapList() {
	g.grid = widget.NewGridWrapList(
		func() int {
			return g.lenItems()
		},
		// create func
		func() fyne.CanvasObject {
			card := NewGridViewCard(g.showYear)
			card.OnPlay = func() {
				if g.OnPlay != nil {
					g.OnPlay(card.ItemID())
				}
			}
			card.OnShowSecondaryPage = func() {
				if g.OnShowSecondaryPage != nil {
					g.OnShowSecondaryPage(card.SecondaryID())
				}
			}
			card.OnShowItemPage = func() {
				if g.OnShowItemPage != nil {
					g.OnShowItemPage(card.ItemID())
				}
			}
			return card
		},
		// update func
		func(itemID int, obj fyne.CanvasObject) {
			ac := obj.(*GridViewCard)
			g.doUpdateItemCard(itemID, ac)
		},
	)
}

func (g *GridView) doUpdateItemCard(itemIdx int, card *GridViewCard) {
	if itemIdx > g.highestShown {
		g.highestShown = itemIdx
	}
	g.itemsMutex.RLock()
	album := g.items[itemIdx]
	g.itemsMutex.RUnlock()
	if card.PrevID == album.ID {
		// nothing to do
		return
	}
	card.Update(album)
	card.PrevID = album.ID
	// cancel any previous image fetch
	if card.ImgLoadCancel != nil {
		card.ImgLoadCancel()
		card.ImgLoadCancel = nil
	}
	if img, ok := g.imageFetcher.GetCoverThumbnailFromCache(album.CoverArt); ok {
		card.Cover.SetImage(img)
	} else {
		card.Cover.SetImageResource(res.ResAlbumplaceholderPng)
		// asynchronously fetch cover image
		ctx, cancel := context.WithCancel(context.Background())
		card.ImgLoadCancel = cancel
		go func(ctx context.Context) {
			i, err := g.imageFetcher.GetCoverThumbnail(album.CoverArt)
			select {
			case <-ctx.Done():
				return
			default:
				if err == nil {
					card.Cover.SetImage(i)
				} else {
					log.Printf("error fetching image: %s", err.Error())
				}
			}
		}(ctx)
	}

	// if user has scrolled near the bottom, fetch more
	if !g.done && !g.fetching && itemIdx > g.lenItems()-10 {
		g.fetchMoreItems(20)
	}
}

func (g *GridView) lenItems() int {
	g.itemsMutex.RLock()
	defer g.itemsMutex.RUnlock()
	return len(g.items)
}

// fetches at least count more items
func (g *GridView) fetchMoreItems(count int) {
	if g.iter == nil {
		g.done = true
	}
	g.fetching = true
	go func() {
		// keep repeating the fetch task as long as the user
		// has scrolled near the bottom
		for !g.done && g.highestShown >= g.lenItems()-10 {
			n := 0
			for !g.done && n < count {
				albums := g.iter.NextN(albumFetchBatchSize)
				g.itemsMutex.Lock()
				g.items = append(g.items, albums...)
				g.itemsMutex.Unlock()
				if len(albums) < albumFetchBatchSize {
					g.done = true
				}
				n += len(albums)
				if len(albums) > 0 {
					g.Refresh()
				}
			}
		}
		g.fetching = false
	}()
}

func (g *GridView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.grid)
}
