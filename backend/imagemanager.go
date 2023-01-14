package backend

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/20after4/configdir"
	"github.com/bluele/gcache"
)

type ImageManager struct {
	s              *ServerManager
	baseCacheDir   string
	thumbnailCache gcache.Cache
}

func NewImageManager(s *ServerManager, baseCacheDir string) *ImageManager {
	cache := gcache.New(100).LRU().Build()
	if err := configdir.MakePath(baseCacheDir); err != nil {
		log.Println("failed to create album cover cache dir")
		baseCacheDir = ""
	}
	return &ImageManager{
		s:              s,
		baseCacheDir:   baseCacheDir,
		thumbnailCache: cache,
	}
}

func (i *ImageManager) GetAlbumThumbnailFromCache(albumID string) (image.Image, bool) {
	if img, err := i.thumbnailCache.Get(albumID); err == nil && img != nil {
		return img.(image.Image), true
	}
	return nil, false
}

func (i *ImageManager) GetAlbumThumbnail(albumID string) (image.Image, error) {
	// in-memory cache
	if i.thumbnailCache.Has(albumID) {
		if img, err := i.thumbnailCache.Get(albumID); err == nil {
			return img.(image.Image), nil
		}
	}

	// on disc cache
	path := i.filePathForCover(albumID)
	if i.ensureCoverCacheDir() != "" {
		if s, err := os.Stat(path); err == nil {
			go i.checkRefreshLocalCover(s, albumID)
			if f, err := os.Open(path); err == nil {
				defer f.Close()
				if img, _, err := image.Decode(f); err == nil {
					i.thumbnailCache.Set(albumID, img)
					return img, nil
				}
			}
		}
	}

	return i.fetchAndCacheCoverFromServer(albumID)
}

func (i *ImageManager) ensureCoverCacheDir() string {
	path := path.Join(i.baseCacheDir, i.s.ServerID.String(), "covers")
	configdir.MakePath(path)
	return path
}

func (i *ImageManager) fetchAndCacheCoverFromServer(albumID string) (image.Image, error) {
	img, err := i.s.Server.GetCoverArt(albumID, map[string]string{"size": "300"})
	if err != nil {
		return nil, err
	}
	if i.ensureCoverCacheDir() != "" {
		path := i.filePathForCover(albumID)
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

func (i *ImageManager) checkRefreshLocalCover(stat os.FileInfo, albumID string) {
	if time.Now().Sub(stat.ModTime()) > 24*time.Hour {
		i.fetchAndCacheCoverFromServer(albumID)
	}
}

func (i *ImageManager) filePathForCover(albumID string) string {
	return filepath.Join(i.ensureCoverCacheDir(), fmt.Sprintf("%s.jpg", albumID))
}
