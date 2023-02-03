package backend

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/20after4/configdir"
)

type ImageManager struct {
	s              *ServerManager
	baseCacheDir   string
	thumbnailCache ImageCache

	cachedFullSizeCover   image.Image
	cachedFullSizeCoverID string
}

func NewImageManager(ctx context.Context, s *ServerManager, baseCacheDir string) *ImageManager {
	if err := configdir.MakePath(baseCacheDir); err != nil {
		log.Println("failed to create album cover cache dir")
		baseCacheDir = ""
	}
	i := &ImageManager{
		s:            s,
		baseCacheDir: baseCacheDir,
		thumbnailCache: ImageCache{
			MinSize:    24,
			MaxSize:    150,
			DefaultTTL: 1 * time.Minute,
		},
	}
	i.thumbnailCache.Init(ctx, 2*time.Minute)
	return i
}

func (i *ImageManager) GetAlbumThumbnailFromCache(albumID string) (image.Image, bool) {
	img, err := i.thumbnailCache.GetExtendTTL(albumID, i.thumbnailCache.DefaultTTL)
	if err == nil && img != nil {
		return img, true
	}
	return nil, false
}

func (i *ImageManager) GetAlbumThumbnail(albumID string) (image.Image, error) {
	if im, ok := i.GetAlbumThumbnailFromCache(albumID); ok {
		return im, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(albumID, i.thumbnailCache.DefaultTTL)
}

func (i *ImageManager) GetAlbumThumbnailWithTTL(albumID string, ttl time.Duration) (image.Image, error) {
	// in-memory cache
	if img, err := i.thumbnailCache.GetWithNewTTL(albumID, ttl); err == nil {
		return img, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(albumID, ttl)
}

func (i *ImageManager) GetFullSizeAlbumCover(albumID string) (image.Image, error) {
	if i.cachedFullSizeCoverID == albumID {
		return i.cachedFullSizeCover, nil
	}
	im, err := i.s.Server.GetCoverArt(albumID, nil)
	if err != nil {
		return nil, err
	}
	i.cachedFullSizeCover = im
	i.cachedFullSizeCoverID = albumID
	return im, nil
}

func (i *ImageManager) ensureCoverCacheDir() string {
	path := path.Join(i.baseCacheDir, i.s.ServerID.String(), "covers")
	configdir.MakePath(path)
	return path
}

func (i *ImageManager) fetchAndCacheCoverFromDiskOrServer(albumID string, ttl time.Duration) (image.Image, error) {
	// on disc cache
	path := i.filePathForCover(albumID)
	if i.ensureCoverCacheDir() != "" {
		if s, err := os.Stat(path); err == nil {
			go i.checkRefreshLocalCover(s, albumID)
			if f, err := os.Open(path); err == nil {
				defer f.Close()
				if img, _, err := image.Decode(f); err == nil {
					i.thumbnailCache.SetWithTTL(albumID, img, ttl)
					return img, nil
				}
			}
		}
	}

	return i.fetchAndCacheCoverFromServer(albumID, ttl)
}

func (i *ImageManager) fetchAndCacheCoverFromServer(albumID string, ttl time.Duration) (image.Image, error) {
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
	if time.Since(stat.ModTime()) > 24*time.Hour {
		i.fetchAndCacheCoverFromServer(albumID, i.thumbnailCache.DefaultTTL)
	}
}

func (i *ImageManager) filePathForCover(albumID string) string {
	return filepath.Join(i.ensureCoverCacheDir(), fmt.Sprintf("%s.jpg", albumID))
}
