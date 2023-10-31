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

const (
	coverArtThumbnailSize = 300
	cachedImageValidTime  = 24 * time.Hour
	fullSizeCoverExpires  = 5 * time.Minute

	maxConcurrentServerFetches = 5
	defaultDiskCacheSizeBytes  = 50 * 1_048_576
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

	serverFetchSema chan interface{}
}

// NewImageManager returns a new ImageManager.
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
		serverFetchSema:         make(chan interface{}, maxConcurrentServerFetches),
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

// SetMaxOnDiskCacheSizeBytes sets the maximum size of the on-disc cover thumbnail cache.
// A periodic clean task will delete least recently accessed images to maintain the size limit.
func (i *ImageManager) SetMaxOnDiskCacheSizeBytes(size int64) {
	i.maxOnDiskCacheSizeBytes = size
}

// GetCoverThumbnailFromCache returns the cover thumbnail for the given ID if it exists
// in the in-memory cache. Returns quickly, safe to call in UI threads.
func (i *ImageManager) GetCoverThumbnailFromCache(coverID string) (image.Image, bool) {
	img, err := i.thumbnailCache.GetExtendTTL(coverID, i.thumbnailCache.DefaultTTL)
	if err == nil && img != nil {
		return img, true
	}
	return nil, false
}

// GetCoverThumbnailAsync asynchronously fetches the cover image for the given ID,
// and invokes the callback on completion. It returns a context.CancelFunc which can be used to
// cancel the fetch. The callback will not be invoked if the fetch is cancelled before completion.
// The cancel func must be invoked to avoid resource leaks. Use GetCoverThumbnail if cancellation is not needed.
func (i *ImageManager) GetCoverThumbnailAsync(coverID string, cb func(image.Image, error)) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if im, ok := i.GetCoverThumbnailFromCache(coverID); ok {
			if ctx.Err() == nil {
				cb(im, nil)
			}
		} else {
			i.fetchAndCacheCoverFromDiskOrServer(ctx, coverID, i.thumbnailCache.DefaultTTL, cb)
		}
	}()
	return cancel
}

// GetCoverThumbnail is a synchronous, blocking function to fetch the image for a given coverID.
// Like most ImageManager calls, it should usually be called in a goroutine to not block UI loading.
func (i *ImageManager) GetCoverThumbnail(coverID string) (image.Image, error) {
	if im, ok := i.GetCoverThumbnailFromCache(coverID); ok {
		return im, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(context.Background(), coverID, i.thumbnailCache.DefaultTTL, nil)
}

// GetCoverThumbnailWithTTL fetches the cover for the given coverID and updates the TTL
// in the in-memory image cache. It blocks until the image fetch is complete.
func (i *ImageManager) GetCoverThumbnailWithTTL(coverID string, ttl time.Duration) (image.Image, error) {
	if img, err := i.thumbnailCache.GetWithNewTTL(coverID, ttl); err == nil {
		return img, nil
	}
	return i.fetchAndCacheCoverFromDiskOrServer(context.Background(), coverID, ttl, nil)
}

// GetFullSizeCoverArt fetches the full size cover image for the given coverID.
// It blocks until the fetch is complete.
func (i *ImageManager) GetFullSizeCoverArt(coverID string) (image.Image, error) {
	if i.cachedFullSizeCoverID == coverID {
		i.cachedFullSizeCoverAccessedAt = time.Now().UnixMilli()
		return i.cachedFullSizeCover, nil
	}
	if i.s.Server == nil {
		return nil, errors.New("logged out")
	}

	i.serverFetchSema <- struct{}{} // acquire
	im, err := i.s.Server.GetCoverArt(coverID, 0)
	<-i.serverFetchSema // release
	if err != nil {
		return nil, err
	}
	i.cachedFullSizeCover = im
	i.cachedFullSizeCoverID = coverID
	i.cachedFullSizeCoverAccessedAt = time.Now().UnixMilli()
	return im, nil
}

// GetCoverArtURL returns the URL for the locally cached cover thumbnail, if it exists.
func (i *ImageManager) GetCoverArtUrl(coverID string) (string, error) {
	path := i.filePathForCover(coverID)
	if _, err := os.Stat(path); err == nil {
		// this is probably broken for Windows but it's currently only used
		// for MPRIS, so we are OK for now
		return fmt.Sprintf("file://%s", path), nil
	}
	return "", errors.New("cover not found")
}

// GetCachedArtistImage returns the artist image for the given artistID from the on-disc cache, if it exists.
func (i *ImageManager) GetCachedArtistImage(artistID string) (image.Image, bool) {
	return i.loadLocalImage(i.filePathForArtistImage(artistID))
}

// FetchAndCacheArtistImage fetches the artist image for the given artistID from the server,
// caching it locally if the fetch succeeds. Blocks until fetch is completed.
func (i *ImageManager) FetchAndCacheArtistImage(artistID string, imgURL string) (image.Image, error) {
	im, err := i.fetchRemoteArtistImage(imgURL)
	if err != nil {
		return nil, err
	}
	_ = i.writeJpeg(im, i.filePathForArtistImage(artistID))
	return im, nil
}

// RefreshCachedArtistImageIfExpired re-fetches the artist image from the server if expired.
func (i *ImageManager) RefreshCachedArtistImageIfExpired(artistID string, imgURL string) error {
	stat, err := os.Stat(i.filePathForArtistImage(artistID))
	if err == nil && time.Since(stat.ModTime()) > cachedImageValidTime {
		_, err = i.FetchAndCacheArtistImage(artistID, imgURL)
	}
	return err
}

// ThumbnailLoader is a utility type that exposes a single API to load
// a cover thumbnail by ID. If the image is immediately available in
// the cache, OnLoaded will be called immediately. If it is not,
// OnBeforeLoad will be called first, then OnLoaded will be called async
// once the image is available.
// Any subsequent calls to Load will cancel the previous load if not yet completed.
type ThumbnailLoader struct {
	prevLoadCancel context.CancelFunc
	im             *ImageManager

	OnBeforeLoad func()
	OnLoaded     func(image.Image)
}

func (i *ImageManager) NewThumbnailLoader(onLoaded func(image.Image)) ThumbnailLoader {
	return ThumbnailLoader{im: i, OnLoaded: onLoaded}
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
			i.callOnLoaded(img)
		}
		i.prevLoadCancel() // Done. Release resources associated with un-cancelled ctx
	})
}

func (i *ThumbnailLoader) callOnLoaded(im image.Image) {
	if i.OnLoaded != nil {
		i.OnLoaded(im)
	}
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
	i.serverFetchSema <- struct{}{} // acquire
	res, err := fyne.LoadResourceFromURLString(url)
	<-i.serverFetchSema // release

	if err == nil {
		im, _, err := image.Decode(bytes.NewReader(res.Content()))
		if err == nil {
			return im, nil
		}
		return nil, err
	}
	return nil, err
}

func (i *ImageManager) fetchAndCacheCoverFromDiskOrServer(ctx context.Context, coverID string, ttl time.Duration, cb func(image.Image, error)) (image.Image, error) {
	// on disc cache
	if i.ensureCoverCacheDir() != "" {
		path := i.filePathForCover(coverID)
		if s, err := os.Stat(path); err == nil {
			go i.checkRefreshLocalCover(s, coverID, ttl)
			if img, ok := i.loadLocalImage(path); ok {
				i.thumbnailCache.SetWithTTL(coverID, img, ttl)
				if ctx.Err() == nil && cb != nil {
					cb(img, nil)
				}
				return img, nil
			}
		}
	}

	// fetch from server
	return i.fetchAndCacheCoverFromServer(ctx, coverID, ttl, cb)
}

func (i *ImageManager) fetchAndCacheCoverFromServer(ctx context.Context, coverID string, ttl time.Duration, cb func(image.Image, error)) (image.Image, error) {
	if i.s.Server == nil {
		err := errors.New("logged out")
		if ctx.Err() == nil && cb != nil {
			cb(nil, err)
		}
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case i.serverFetchSema <- struct{}{}: // acquire
		img, err := i.s.Server.GetCoverArt(coverID, coverArtThumbnailSize)
		<-i.serverFetchSema // release
		if err == nil {
			if i.ensureCoverCacheDir() != "" {
				_ = i.writeJpeg(img, i.filePathForCover(coverID))
			}
			i.thumbnailCache.SetWithTTL(coverID, img, ttl)
		}
		if ctx.Err() == nil && cb != nil {
			cb(img, err)
		}
		return img, err
	}
}

func (i *ImageManager) checkRefreshLocalCover(stat os.FileInfo, coverID string, ttl time.Duration) {
	if time.Since(stat.ModTime()) > cachedImageValidTime {
		i.fetchAndCacheCoverFromServer(context.Background(), coverID, ttl, nil)
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
