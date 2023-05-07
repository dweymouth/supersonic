package widgets

import (
	"context"
	"image"
	"log"
	"sync"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

const batchFetchSize = 6

type ImageFetcher interface {
	GetCoverThumbnailFromCache(string) (image.Image, bool)
	GetCoverThumbnail(string) (image.Image, error)
}

type GridViewIterator interface {
	NextN(int) []GridViewItemModel
}

type gridViewAlbumIterator struct {
	iter *backend.BatchingIterator
}

func (g gridViewAlbumIterator) NextN(n int) []GridViewItemModel {
	albums := g.iter.NextN(n)
	return sharedutil.MapSlice(albums, func(al *subsonic.AlbumID3) GridViewItemModel {
		return GridViewItemModel{
			Name:        al.Name,
			ID:          al.ID,
			CoverArtID:  al.CoverArt,
			Secondary:   al.Artist,
			SecondaryID: al.ArtistID,
		}
	})
}

func NewGridViewAlbumIterator(iter backend.AlbumIterator) GridViewIterator {
	return gridViewAlbumIterator{iter: backend.NewBatchingIterator(iter)}
}

type GridView struct {
	widget.BaseWidget

	GridViewState

	grid *GridWrap
}

type GridViewState struct {
	items        []GridViewItemModel
	itemsMutex   sync.RWMutex
	iter         GridViewIterator
	imageFetcher ImageFetcher
	highestShown int
	fetching     bool
	done         bool

	OnPlay              func(id string, shuffle bool)
	OnAddToQueue        func(id string)
	OnAddToPlaylist     func(id string)
	OnShowItemPage      func(id string)
	OnShowSecondaryPage func(id string)

	scrollPos float32
}

var _ fyne.Widget = (*GridView)(nil)

func NewFixedGridView(items []GridViewItemModel, fetch ImageFetcher) *GridView {
	g := &GridView{
		GridViewState: GridViewState{
			items:        items,
			done:         true,
			imageFetcher: fetch,
		},
	}
	g.ExtendBaseWidget(g)
	g.createGridWrap()
	return g
}

func NewGridView(iter GridViewIterator, fetch ImageFetcher) *GridView {
	g := &GridView{
		GridViewState: GridViewState{
			iter:         iter,
			imageFetcher: fetch,
		},
	}
	g.ExtendBaseWidget(g)

	g.createGridWrap()

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
	g.createGridWrap()
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

func (g *GridView) Reset(iter GridViewIterator) {
	g.itemsMutex.Lock()
	g.items = nil
	g.itemsMutex.Unlock()
	g.fetching = false
	g.done = false
	g.highestShown = 0
	g.iter = iter
	g.fetchMoreItems(36)
}

func (g *GridView) ResetFixed(items []GridViewItemModel) {
	g.itemsMutex.Lock()
	g.items = items
	g.itemsMutex.Unlock()
	g.fetching = false
	g.done = true
	g.highestShown = 0
	g.iter = nil
}

func (g *GridView) createGridWrap() {
	g.grid = NewGridWrap(
		func() int {
			return g.lenItems()
		},
		// create func
		func() fyne.CanvasObject {
			card := NewGridViewItem()
			card.OnPlay = func(shuffle bool) {
				if g.OnPlay != nil {
					g.OnPlay(card.ItemID(), shuffle)
				}
			}
			card.OnAddToQueue = func() {
				if g.OnAddToQueue != nil {
					g.OnAddToQueue(card.ItemID())
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
			card.OnAddToPlaylist = func() {
				if g.OnAddToPlaylist != nil {
					g.OnAddToPlaylist(card.ItemID())
				}
			}
			return card
		},
		// update func
		func(itemID GridWrapItemID, obj fyne.CanvasObject) {
			ac := obj.(*GridViewItem)
			g.doUpdateItemCard(int(itemID), ac)
		},
	)
}

func (g *GridView) doUpdateItemCard(itemIdx int, card *GridViewItem) {
	if itemIdx > g.highestShown {
		g.highestShown = itemIdx
	}
	g.itemsMutex.RLock()
	item := g.items[itemIdx]
	g.itemsMutex.RUnlock()
	if card.PrevID == item.ID {
		// nothing to do
		return
	}
	card.Update(item)
	card.PrevID = item.ID
	// cancel any previous image fetch
	if card.ImgLoadCancel != nil {
		card.ImgLoadCancel()
		card.ImgLoadCancel = nil
	}
	if img, ok := g.imageFetcher.GetCoverThumbnailFromCache(item.CoverArtID); ok {
		card.Cover.SetImage(img)
	} else {
		card.Cover.SetImageResource(res.ResAlbumplaceholderPng)
		// asynchronously fetch cover image
		ctx, cancel := context.WithCancel(context.Background())
		card.ImgLoadCancel = cancel
		go func(ctx context.Context) {
			i, err := g.imageFetcher.GetCoverThumbnail(item.CoverArtID)
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
				items := g.iter.NextN(batchFetchSize)
				g.itemsMutex.Lock()
				g.items = append(g.items, items...)
				g.itemsMutex.Unlock()
				if len(items) < batchFetchSize {
					g.done = true
				}
				n += len(items)
				if len(items) > 0 {
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
