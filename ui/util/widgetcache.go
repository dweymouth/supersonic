package util

import (
	"time"

	"fyne.io/fyne/v2"
)

type WidgetType string

const (
	WidgetTypeAlbumPageHeader  WidgetType = "AlbumPageHeader"
	WidgetTypeArtistPageHeader WidgetType = "ArtistPageHeader"
)

type WidgetCache struct {
	cache map[WidgetType][]cachedWidget
}

type cachedWidget struct {
	widget     fyne.CanvasObject
	releasedAt int64 // unixMillis
}

func NewWidgetCache() WidgetCache {
	return WidgetCache{
		cache: make(map[WidgetType][]cachedWidget),
	}
}

func (w *WidgetCache) Obtain(typ WidgetType) fyne.CanvasObject {
	var widget fyne.CanvasObject
	if ws, ok := w.cache[typ]; ok && len(ws) > 0 {
		i := len(ws) - 1
		widget = ws[i].widget
		ws[i].widget = nil
		w.cache[typ] = ws[:i]
	}
	return widget
}

func (w *WidgetCache) Release(typ WidgetType, wid fyne.CanvasObject) {
	if _, ok := w.cache[typ]; !ok {
		w.cache[typ] = make([]cachedWidget, 0)
	}
	w.cache[typ] = append(w.cache[typ], cachedWidget{
		widget:     wid,
		releasedAt: time.Now().UnixMilli(),
	})
}
