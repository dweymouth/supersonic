package backend

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"

	"github.com/20after4/configdir"
	"github.com/bluele/gcache"
	subsonic "github.com/dweymouth/go-subsonic"
)

type ImageManager struct {
	s              *subsonic.Client
	cacheDir       string
	thumbnailCache gcache.Cache
}

func NewImageManager(s *subsonic.Client, cacheDir string) *ImageManager {
	cache := gcache.New(100).LRU().Build()
	if err := configdir.MakePath(cacheDir); err != nil {
		log.Println("failed to create album cover cache dir")
		cacheDir = ""
	}
	return &ImageManager{
		s:              s,
		cacheDir:       cacheDir,
		thumbnailCache: cache,
	}
}

func (i *ImageManager) GetAlbumThumbnail(albumID string) (image.Image, error) {
	// in-memory cache
	if i.thumbnailCache.Has(albumID) {
		if img, err := i.thumbnailCache.Get(albumID); err == nil {
			return img.(image.Image), nil
		}
	}

	// on disc cache
	path := filepath.Join(i.cacheDir, fmt.Sprintf("%s.jpg", albumID))
	if i.cacheDir != "" {
		if _, err := os.Stat(path); err == nil {
			// serve image from on-disc cache
			// TODO: image may have changed on server.
			//    first, return cached image, then fetch fresh img from server in background
			if f, err := os.Open(path); err == nil {
				defer f.Close()
				if img, _, err := image.Decode(f); err == nil {
					i.thumbnailCache.Set(albumID, img)
					return img, nil
				}
			}
		}
	}

	// fetch from server
	img, err := i.s.GetCoverArt(albumID, map[string]string{"size": "300"})
	if err != nil {
		return nil, err
	}
	if i.cacheDir != "" {
		if f, err := os.Create(path); err == nil {
			defer f.Close()
			if err := jpeg.Encode(f, img, nil /*options*/); err != nil {
				log.Printf("failed to cache image: %s", err.Error())
			}
		}
	}
	i.thumbnailCache.Set(albumID, img)
	return img, nil
}
