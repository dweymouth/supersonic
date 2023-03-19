package backend

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"github.com/20after4/configdir"
)

const CachedImageValidTime = 24 * time.Hour

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

func (i *ImageManager) GetCachedArtistImage(artistID string) (image.Image, bool) {
	return i.loadLocalImage(i.filePathForArtistImage(artistID))
}

func (i *ImageManager) FetchAndCacheArtistImage(artistID string, imgURL string) (image.Image, error) {
	im, err := i.fetchRemoteArtistImage(imgURL)
	if err != nil {
		return nil, err
	}
	_ = i.writeJpeg(im, i.filePathForArtistImage(artistID))
	return im, nil
}

func (i *ImageManager) RefreshCachedArtistImageIfExpired(artistID string, imgURL string) error {
	stat, err := os.Stat(i.filePathForArtistImage(artistID))
	if err == nil && time.Since(stat.ModTime()) > CachedImageValidTime {
		_, err = i.FetchAndCacheArtistImage(artistID, imgURL)
	}
	return err
}

func (i *ImageManager) ensureCoverCacheDir() string {
	path := path.Join(i.baseCacheDir, i.s.ServerID.String(), "covers")
	configdir.MakePath(path)
	return path
}

func (i *ImageManager) ensureArtistCoverCacheDir() string {
	path := path.Join(i.baseCacheDir, i.s.ServerID.String(), "artistimages")
	configdir.MakePath(path)
	return path
}

func (i *ImageManager) fetchRemoteArtistImage(url string) (image.Image, error) {
	res, err := fyne.LoadResourceFromURLString(url)
	if err == nil {
		im, _, err := image.Decode(bytes.NewReader(res.Content()))
		if err == nil {
			return im, nil
		}
		return nil, err
	}
	return nil, err
}

func (i *ImageManager) fetchAndCacheCoverFromDiskOrServer(albumID string, ttl time.Duration) (image.Image, error) {
	// on disc cache
	path := i.filePathForCover(albumID)
	if i.ensureCoverCacheDir() != "" {
		if s, err := os.Stat(path); err == nil {
			go i.checkRefreshLocalCover(s, albumID, ttl)
			if img, ok := i.loadLocalImage(path); ok {
				i.thumbnailCache.SetWithTTL(albumID, img, ttl)
				return img, nil
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
		_ = i.writeJpeg(img, path)
	}
	i.thumbnailCache.SetWithTTL(albumID, img, ttl)
	return img, nil
}

func (i *ImageManager) checkRefreshLocalCover(stat os.FileInfo, albumID string, ttl time.Duration) {
	if time.Since(stat.ModTime()) > CachedImageValidTime {
		i.fetchAndCacheCoverFromServer(albumID, ttl)
	}
}

func (i *ImageManager) filePathForCover(albumID string) string {
	return filepath.Join(i.ensureCoverCacheDir(), fmt.Sprintf("%s.jpg", albumID))
}

func (i *ImageManager) filePathForArtistImage(id string) string {
	return filepath.Join(i.ensureArtistCoverCacheDir(), fmt.Sprintf("%s.jpg", id))
}

func (i *ImageManager) writeJpeg(img image.Image, path string) error {
	f, err := os.Create(path)
	if err == nil {
		defer f.Close()
		if err := jpeg.Encode(f, img, nil /*options*/); err != nil {
			log.Printf("failed to cache image: %s", err.Error())
			return err
		}
	}
	return err
}

func (i *ImageManager) loadLocalImage(path string) (image.Image, bool) {
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		if img, _, err := image.Decode(f); err == nil {
			return img, true
		}
	}
	return nil, false
}
