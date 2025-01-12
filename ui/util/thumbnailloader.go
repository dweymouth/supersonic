package util

import (
	"context"
	"image"
	"log"

	"fyne.io/fyne/v2"
)

// ThumbnailLoader is a utility type that exposes a single API to load
// a cover thumbnail by ID. If the image is immediately available in
// the cache, OnLoaded will be called immediately. If it is not,
// OnBeforeLoad will be called first, then OnLoaded will be called async
// once the image is available.
// Any subsequent calls to Load will cancel the previous load if not yet completed.
type ThumbnailLoader struct {
	prevLoadCancel context.CancelFunc
	im             ImageFetcher

	OnBeforeLoad func()
	OnLoaded     func(image.Image)
}

// Image backend interface for the ThumbnailLoader
// impl: backend.ImageManager
type ImageFetcher interface {
	GetCoverThumbnailFromCache(string) (image.Image, bool)
	GetCoverThumbnailAsync(string, func(image.Image, error)) context.CancelFunc
}

func NewThumbnailLoader(im ImageFetcher, onLoaded func(image.Image)) ThumbnailLoader {
	return ThumbnailLoader{im: im, OnLoaded: onLoaded}
}

func (i *ThumbnailLoader) Load(coverID string) {
	if i.prevLoadCancel != nil {
		i.prevLoadCancel()
	}
	if coverID == "" {
		i.callOnLoaded(nil)
		return
	}
	if img, ok := i.im.GetCoverThumbnailFromCache(coverID); ok {
		i.callOnLoaded(img)
		return
	}
	if i.OnBeforeLoad != nil {
		i.OnBeforeLoad()
	}
	i.prevLoadCancel = i.im.GetCoverThumbnailAsync(coverID, func(img image.Image, err error) {
		if err != nil {
			log.Printf("Error loading cover image: %s", err.Error())
		} else {
			fyne.Do(func() { i.callOnLoaded(img) })
		}
		i.prevLoadCancel() // Done. Release resources associated with un-cancelled ctx
	})
}

func (i *ThumbnailLoader) callOnLoaded(im image.Image) {
	if i.OnLoaded != nil {
		i.OnLoaded(im)
	}
}
