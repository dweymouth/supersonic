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
//  1. If there are fewer than MinSize items in the cache, none will be evicted
//  2. If a new addition would make the cache exceed MaxSize, an item will be immediately evicted
//     2a. in this case, evict the LRU expired item or if none expired, the LRU item
//  3. If the size of the cache is between MaxSize and MinSize, expired items will be periodically evicted
//     3a. in this case, again the least recently used expired items will be evicted first
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

// holds writer lock for O(i.MaxSize) worst case
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

func (i *ImageCache) GetWithNewTTL(key string, newTtl time.Duration) (image.Image, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		v.expiresAt = time.Now().Add(newTtl).Unix()
		v.ttl = newTtl
		return v.val, nil
	}
	return nil, ErrNotFound
}

// must be called when rwmutex is already acquired for writing
func (i *ImageCache) evictOne() {
	now := time.Now().Unix()
	var lruKey string
	lruTime := now
	var lruExpiredKey string
	lruExpiredTime := now
	for k, v := range i.cache {
		if v.expiresAt < now && v.lastAccessed < lruExpiredTime {
			lruExpiredTime = v.lastAccessed
			lruExpiredKey = k
		}
		if v.lastAccessed < lruTime {
			lruTime = v.lastAccessed
			lruKey = k
		}
	}
	if lruExpiredTime < now {
		// deleting LRU expired item
		delete(i.cache, lruExpiredKey)
	} else {
		// no expired items, delete LRU non-expired item
		delete(i.cache, lruKey)
	}
}

type expiredItem struct {
	key          string
	lastAccessed int64
}

// TODO
func (i *ImageCache) EvictExpired() {
	i.mu.RLock()
	expired := make([]expiredItem, 0, len(i.cache)-i.MinSize)
	now := time.Now().Unix()
	for k, v := range i.cache {
		if v.expiresAt < now {
			expired = append(expired, expiredItem{key: k, lastAccessed: v.lastAccessed})
		}
	}
	i.mu.RUnlock()

	count := len(i.cache)
	for count > i.MinSize {

	}

}

func heapify(arr *[]expiredItem, i int) {
	//smallest := i
	//lChild := 2*i + 1
	//rChild := 2*i + 2

}
