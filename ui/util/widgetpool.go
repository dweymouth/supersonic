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

	// keep at bottom
	numWidgetTypes
)

// A pool to share commonly-used widgets across pages to reduce
// creation of new widgets and memory allocations.
// It is not thread-safe, which is fine for its current use.
type WidgetPool struct {
	pool [][]pooledWidget
}

type pooledWidget struct {
	widget     fyne.CanvasObject
	releasedAt int64 // unixMillis
}

func NewWidgetPool() WidgetPool {
	return WidgetPool{
		pool: make([][]pooledWidget, numWidgetTypes),
	}
}

// Obtain obtains a widget of the given type from the pool, if one exists.
// Returns nil if there is no available widget.
func (w *WidgetPool) Obtain(typ WidgetType) fyne.CanvasObject {
	var widget fyne.CanvasObject
	if l := len(w.pool[typ]); l > 0 {
		i := l - 1
		widget = w.pool[typ][i].widget
		w.pool[typ][i].widget = nil
		w.pool[typ] = w.pool[typ][:i]
	}
	return widget
}

// Release releases a widget into the pool.
// The widget must not be modified by the releaser after release,
// since it may be Obtained for a new use at any time.
func (w *WidgetPool) Release(typ WidgetType, wid fyne.CanvasObject) {
	w.pool[typ] = append(w.pool[typ], pooledWidget{
		widget:     wid,
		releasedAt: time.Now().UnixMilli(),
	})
}
