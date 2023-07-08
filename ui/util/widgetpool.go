package util

import (
	"time"

	"fyne.io/fyne/v2"
)

type WidgetType int

const (
	WidgetTypeAlbumPageHeader WidgetType = iota
	WidgetTypeArtistPageHeader
	WidgetTypePlaylistPageHeader
	WidgetTypeTracklist
)

// A pool to share commonly-used widgets across pages to reduce
// creation of new widgets and memory allocations.
// It is not thread-safe, which is fine for its current use.
type WidgetPool struct {
	cache map[WidgetType][]cachedWidget
}

type cachedWidget struct {
	widget     fyne.CanvasObject
	releasedAt int64 // unixMillis
}

func NewWidgetPool() WidgetPool {
	return WidgetPool{
		cache: make(map[WidgetType][]cachedWidget),
	}
}

// Obtain obtains a widget of the given type from the pool, if one exists.
// Returns nil if there is no available widget.
func (w *WidgetPool) Obtain(typ WidgetType) fyne.CanvasObject {
	var widget fyne.CanvasObject
	if ws, ok := w.cache[typ]; ok && len(ws) > 0 {
		i := len(ws) - 1
		widget = ws[i].widget
		ws[i].widget = nil
		w.cache[typ] = ws[:i]
	}
	return widget
}

// Release releases a widget into the pool.
// The widget must not be modified by the releaser after release,
// since it may be Obtained for a new use at any time.
func (w *WidgetPool) Release(typ WidgetType, wid fyne.CanvasObject) {
	if _, ok := w.cache[typ]; !ok {
		w.cache[typ] = make([]cachedWidget, 0)
	}
	w.cache[typ] = append(w.cache[typ], cachedWidget{
		widget:     wid,
		releasedAt: time.Now().UnixMilli(),
	})
}
