package backend

import (
	"image"

	"github.com/bluele/gcache"
	subsonic "github.com/dweymouth/go-subsonic"
)

type ImageManager struct {
	s              *subsonic.Client
	thumbnailCache gcache.Cache
}

func NewImageManager(s *subsonic.Client) *ImageManager {
	cache := gcache.New(100).LRU().Build()
	return &ImageManager{
		s:              s,
		thumbnailCache: cache,
	}
}

func (i *ImageManager) GetAlbumThumbnail(albumID string) (image.Image, error) {
	if i.thumbnailCache.Has(albumID) {
		if img, err := i.thumbnailCache.Get(albumID); err == nil {
			return img.(image.Image), nil
		}
	}
	// TODO: on disc cache
	img, err := i.s.GetCoverArt(albumID, map[string]string{"size": "250"})
	if err != nil {
		return nil, err
	}
	i.thumbnailCache.Set(albumID, img)
	return img, nil
}
