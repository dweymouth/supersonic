package backend

import (
	"context"
	"errors"
	"image"
	"sync"
	"time"
)

type CacheItem struct {
	val image.Image
	ttl time.Duration

	// unix time
	expiresAt    int64
	lastAccessed int64
}

// A custom cache for images with the following eviction strategy:
//   - If there are fewer than MinSize items in the cache, none will be evicted
//   - If a new addition would make the cache exceed MaxSize, an item will be immediately evicted
//   - evicted item will be an expired item if possible, else the least recently used
//   - If the size of the cache is between MaxSize and MinSize, expired items will be periodically evicted
type ImageCache struct {
	MinSize    int
	MaxSize    int
	DefaultTTL time.Duration

	mu    sync.RWMutex
	cache map[string]CacheItem
}

var (
	ErrNotFound = errors.New("item not found")
)

func (i *ImageCache) Init(ctx context.Context) {
	i.cache = make(map[string]CacheItem)
}

func (i *ImageCache) AddWithTTL(key string, val image.Image, ttl time.Duration) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if v, ok := i.cache[key]; ok {
		v.val = val
		v.ttl = ttl
		v.expiresAt = time.Now().Add(v.ttl).Unix()
		v.lastAccessed = time.Now().Unix()
		return
	}
	if len(i.cache) == i.MaxSize {
		i.evictOne()
		i.cache[key] = CacheItem{
			val:          val,
			ttl:          ttl,
			expiresAt:    time.Now().Add(ttl).Unix(),
			lastAccessed: time.Now().Unix(),
		}
	}
}

func (i *ImageCache) Add(key string, val image.Image) {
	i.AddWithTTL(key, val, i.DefaultTTL)
}

func (i *ImageCache) Has(key string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	_, ok := i.cache[key]
	return ok
}

func (i *ImageCache) Get(key string, resetTTL bool) (image.Image, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		if resetTTL {
			v.expiresAt = time.Now().Add(v.ttl).Unix()
		}
		return v.val, nil
	}
	return nil, ErrNotFound
}

// assuming max size is small enough that linear scan is fine
func (i *ImageCache) evictOne() {
	var lruKey string
	now := time.Now().Unix()
	lruTime := now
	for k, v := range i.cache {
		if v.expiresAt < now {
			delete(i.cache, k)
			return
		}
		if v.lastAccessed < lruTime {
			lruTime = v.lastAccessed
			lruKey = k
		}
	}
	delete(i.cache, lruKey)
}

// TODO
func (i *ImageCache) EvictExpired() {
	i.mu.Lock()
	defer i.mu.Unlock()

	count := len(i.cache)
	for count > i.MinSize {

	}

}
