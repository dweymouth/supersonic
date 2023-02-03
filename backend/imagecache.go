package backend

import (
	"container/heap"
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

// A custom in-memory cache for images with the following eviction strategy:
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

func (i *ImageCache) Init(ctx context.Context, evictionInterval time.Duration) {
	i.cache = make(map[string]CacheItem)
	go i.periodicallyEvict(ctx, evictionInterval)
}

// holds writer lock for O(i.MaxSize) worst case
func (i *ImageCache) SetWithTTL(key string, val image.Image, ttl time.Duration) {
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
	}
	i.cache[key] = CacheItem{
		val:          val,
		ttl:          ttl,
		expiresAt:    time.Now().Add(ttl).Unix(),
		lastAccessed: time.Now().Unix(),
	}
}

func (i *ImageCache) Set(key string, val image.Image) {
	i.SetWithTTL(key, val, i.DefaultTTL)
}

func (i *ImageCache) Has(key string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	_, ok := i.cache[key]
	return ok
}

func (i *ImageCache) Get(key string) (image.Image, error) {
	return i.GetResetTTL(key, false)
}

func (i *ImageCache) GetResetTTL(key string, resetTTL bool) (image.Image, error) {
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

// Gets the image if it exists and extends TTL to time.Now + ttl iff the image would expire before then
func (i *ImageCache) GetExtendTTL(key string, ttl time.Duration) (image.Image, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		if v.expiresAt < time.Now().Add(ttl).Unix() {
			v.expiresAt = time.Now().Add(ttl).Unix()
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

func (i *ImageCache) periodicallyEvict(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			i.EvictExpired()
		}
	}
}

type expiredItem struct {
	key          string
	lastAccessed int64
}

type expiredHeap []expiredItem

func (h expiredHeap) Len() int           { return len(h) }
func (h expiredHeap) Less(i, j int) bool { return h[i].lastAccessed < h[j].lastAccessed }
func (h expiredHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *expiredHeap) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(expiredItem))
}

func (h *expiredHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// EvictExpired evicts least recently used expired items from the cache
// until there are no more expired items or the cache contains MinSize elements
// Holds the reader lock for O(n) time and writer lock for O(n)
func (i *ImageCache) EvictExpired() {
	i.mu.RLock()
	count := len(i.cache)
	sliceCap := count - i.MinSize
	if sliceCap <= 0 {
		i.mu.RUnlock()
		return
	}
	expired := make(expiredHeap, 0, sliceCap)
	now := time.Now().Unix()
	for k, v := range i.cache {
		if v.expiresAt < now {
			expired = append(expired, expiredItem{key: k, lastAccessed: v.lastAccessed})
		}
	}
	i.mu.RUnlock()

	heap.Init(&expired)
	var keysToRemove []string
	for count > i.MinSize && len(expired) > 0 {
		keysToRemove = append(keysToRemove, heap.Pop(&expired).(expiredItem).key)
		count -= 1
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	for _, key := range keysToRemove {
		// during the interim when we don't hold the lock, some expired items
		// could have been re-set, so check expiry again
		if item, ok := i.cache[key]; ok && item.expiresAt < now {
			delete(i.cache, key)
		}
	}

}
