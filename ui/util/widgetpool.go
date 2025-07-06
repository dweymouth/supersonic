package util

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

type WidgetType int

const (
	WidgetTypeAlbumPageHeader WidgetType = iota
	WidgetTypeArtistPageHeader
	WidgetTypeGridView
	WidgetTypePlaylistPageHeader
	WidgetTypeTracklist
	WidgetTypeCompactTracklist
	WidgetTypeNowPlayingPage
	WidgetTypeGroupedReleases

	// keep at bottom
	numWidgetTypes
)

const (
	multipleItemsExpiry = 2 * time.Minute
	singleItemExpiry    = 5 * time.Minute
)

// A pool to share commonly-used widgets across pages to reduce
// creation of new widgets and memory allocations.
type WidgetPool struct {
	mut   sync.Mutex
	pools [][]pooledWidget
}

type pooledWidget struct {
	widget     fyne.CanvasObject
	releasedAt int64 // unixMillis
}

func NewWidgetPool() *WidgetPool {
	p := &WidgetPool{
		pools: make([][]pooledWidget, numWidgetTypes),
	}
	go func() {
		t := time.NewTicker(2 * time.Minute)
		for range t.C {
			p.cleanUpExpiredItems()
		}
	}()
	return p
}

// Obtain obtains a widget of the given type from the pool, if one exists.
// Returns nil if there is no available widget.
func (w *WidgetPool) Obtain(typ WidgetType) fyne.CanvasObject {
	w.mut.Lock()
	defer w.mut.Unlock()
	var widget fyne.CanvasObject
	if l := len(w.pools[typ]); l > 0 {
		i := l - 1
		widget = w.pools[typ][i].widget
		w.pools[typ][i].widget = nil
		w.pools[typ] = w.pools[typ][:i]
	}
	return widget
}

// Release releases a widget into the pool.
// The widget must not be modified by the releaser after release,
// since it may be Obtained for a new use at any time.
func (w *WidgetPool) Release(typ WidgetType, wid fyne.CanvasObject) {
	w.mut.Lock()
	defer w.mut.Unlock()
	w.pools[typ] = append(w.pools[typ], pooledWidget{
		widget:     wid,
		releasedAt: time.Now().UnixMilli(),
	})
}

func (w *WidgetPool) cleanUpExpiredItems() {
	w.mut.Lock()
	defer w.mut.Unlock()
	for widTyp, pool := range w.pools {
		newP := make([]pooledWidget, 0, len(pool))
		l := len(pool)
		for _, wid := range pool {
			timeSinceRelease := time.Since(time.UnixMilli(wid.releasedAt))
			if l > 1 && timeSinceRelease > multipleItemsExpiry {
				l-- // let expire if >1 item in pool and released long enough ago
			} else if l == 1 && timeSinceRelease > singleItemExpiry {
				l-- // let expire if last item in pool and released long enough ago
			} else {
				newP = append(newP, wid) // not expired
			}
		}
		// re-assign non-expired items back to this widget type pool
		w.pools[widTyp] = newP
	}
}
