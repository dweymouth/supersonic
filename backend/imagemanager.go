package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"github.com/20after4/configdir"
	"github.com/google/uuid"
)

const CachedImageValidTime = 24 * time.Hour

const (
	coverArtThumbnailSize = 300
	fullSizeCoverExpires  = 5 * time.Minute

	defaultDiskCacheSizeBytes = 50 * 1_048_576
)

// The ImageManager is responsible for retrieving and serving images to the UI layer.
// It maintains an in-memory cache of recently used images for immediate future access,
// and a larger on-disc cache of images that is periodically re-requested from the server.
type ImageManager struct {
	s              *ServerManager
	baseCacheDir   string
	thumbnailCache ImageCache

	cachedFullSizeCover           image.Image
	cachedFullSizeCoverID         string
	cachedFullSizeCoverAccessedAt int64 // unixMillis

	maxOnDiskCacheSizeBytes    int64
	filesWrittenSinceLastPrune bool
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
		maxOnDiskCacheSizeBytes: defaultDiskCacheSizeBytes,
	}
	s.OnLogout(func() {
		i.thumbnailCache.Clear()
		i.clearFullSizeCover()
	})
	i.thumbnailCache.OnEvictTaskRan = func() {
		i.clearFullSizeCoverIfExpired()
		i.pruneOnDiskCache()
	}
	i.thumbnailCache.Init(ctx, 2*time.Minute)
	return i
}

func (i *ImageManager) SetMaxOnDiskCacheSizeBytes(size int64) {
	i.maxOnDiskCacheSizeBytes = size
}

func (i *ImageManager) GetCoverThumbnailFromCache(coverID string) (image.Image, bool) {
	img, err := i.thumbnailCache.GetExtendTTL(coverID, i.thumbnailCache.DefaultTTL)
	if err == nil && img != nil {
		return img, true
	}
	return nil, false
}

func (i *ImageManager) GetCoverThumbnail(coverID string) (image.Image, error) {
	if im, ok := i.GetCoverThumbnailFromCache(coverID); ok {
		return im, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(coverID, i.thumbnailCache.DefaultTTL)
}

func (i *ImageManager) GetCoverThumbnailWithTTL(coverID string, ttl time.Duration) (image.Image, error) {
	// in-memory cache
	if img, err := i.thumbnailCache.GetWithNewTTL(coverID, ttl); err == nil {
		return img, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(coverID, ttl)
}

func (i *ImageManager) GetFullSizeCoverArt(coverID string) (image.Image, error) {
	if i.cachedFullSizeCoverID == coverID {
		i.cachedFullSizeCoverAccessedAt = time.Now().UnixMilli()
		return i.cachedFullSizeCover, nil
	}
	if i.s.Server == nil {
		return nil, errors.New("logged out")
	}
	im, err := i.s.Server.GetCoverArt(coverID, 0)
	if err != nil {
		return nil, err
	}
	i.cachedFullSizeCover = im
	i.cachedFullSizeCoverID = coverID
	i.cachedFullSizeCoverAccessedAt = time.Now().UnixMilli()
	return im, nil
}

func (i *ImageManager) GetCoverArtUrl(coverID string) (string, error) {
	path := i.filePathForCover(coverID)
	if _, err := os.Stat(path); err == nil {
		// this is probably broken for Windows but it's currently only used
		// for MPRIS, so we are OK for now
		return fmt.Sprintf("file://%s", path), nil
	}
	return "", errors.New("cover not found")
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
	// if user logged out with pending fetches in progress,
	// make sure we don't write to nil (00000000-*0) cache directory
	if i.s.ServerID == uuid.Nil {
		return ""
	}
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

func (i *ImageManager) fetchAndCacheCoverFromDiskOrServer(coverID string, ttl time.Duration) (image.Image, error) {
	// on disc cache
	path := i.filePathForCover(coverID)
	if i.ensureCoverCacheDir() != "" {
		if s, err := os.Stat(path); err == nil {
			go i.checkRefreshLocalCover(s, coverID, ttl)
			if img, ok := i.loadLocalImage(path); ok {
				i.thumbnailCache.SetWithTTL(coverID, img, ttl)
				return img, nil
			}
		}
	}

	return i.fetchAndCacheCoverFromServer(coverID, ttl)
}

func (i *ImageManager) fetchAndCacheCoverFromServer(coverID string, ttl time.Duration) (image.Image, error) {
	if i.s.Server == nil {
		return nil, errors.New("logged out")
	}
	img, err := i.s.Server.GetCoverArt(coverID, coverArtThumbnailSize)
	if err != nil {
		return nil, err
	}
	if i.ensureCoverCacheDir() != "" {
		path := i.filePathForCover(coverID)
		_ = i.writeJpeg(img, path)
	}
	i.thumbnailCache.SetWithTTL(coverID, img, ttl)
	return img, nil
}

func (i *ImageManager) checkRefreshLocalCover(stat os.FileInfo, coverID string, ttl time.Duration) {
	if time.Since(stat.ModTime()) > CachedImageValidTime {
		i.fetchAndCacheCoverFromServer(coverID, ttl)
	}
}

func (i *ImageManager) filePathForCover(coverID string) string {
	return filepath.Join(i.ensureCoverCacheDir(), fmt.Sprintf("%s.jpg", coverID))
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
	i.filesWrittenSinceLastPrune = true
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

func (i *ImageManager) clearFullSizeCoverIfExpired() {
	now := time.Now().UnixMilli()
	if now-i.cachedFullSizeCoverAccessedAt > fullSizeCoverExpires.Milliseconds() {
		i.clearFullSizeCover()
	}
}

func (i *ImageManager) clearFullSizeCover() {
	i.cachedFullSizeCoverID = ""
	i.cachedFullSizeCover = nil
}

func (im *ImageManager) pruneOnDiskCache() {
	if !im.filesWrittenSinceLastPrune {
		return // no new covers cached since last run, no need to walk dir
	}

	// collect list of all cached covers (across servers)
	// we use modTime as a proxy for last accessed time
	// since covers are refreshed from the server after a fixed interval,
	// modTime is roughly equivalent to last access
	type fileInfo struct {
		path    string
		size    int64
		modTime int64
	}
	var allCovers []fileInfo
	var totalSize int64
	filepath.WalkDir(im.baseCacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "jpg") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			s := info.Size()
			allCovers = append(allCovers,
				fileInfo{path: path, size: s, modTime: info.ModTime().UnixMilli()})
			totalSize += s
		}

		return nil
	})

	if totalSize > im.maxOnDiskCacheSizeBytes {
		// sort and then delete from least recently modified until size is under threshold
		sort.Slice(allCovers, func(i, j int) bool {
			return allCovers[i].modTime < allCovers[j].modTime
		})
		for i := 0; i < len(allCovers) && totalSize > im.maxOnDiskCacheSizeBytes; i++ {
			if err := os.Remove(allCovers[i].path); err == nil {
				totalSize -= allCovers[i].size
			}
		}
	}
	im.filesWrittenSinceLastPrune = false
}
