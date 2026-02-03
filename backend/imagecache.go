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

// ImageCache is a thread-safe in-memory cache for images with LRU eviction.
//
// Eviction strategy:
//  1. If there are fewer than MinSize items in the cache, none will be evicted
//  2. If a new addition would make the cache exceed MaxSize, an item will be immediately evicted
//     2a. in this case, evict the LRU expired item or if none expired, the LRU item
//  3. If the size of the cache is between MaxSize and MinSize, expired items will be periodically evicted
//     3a. in this case, again the least recently used expired items will be evicted first
type ImageCache struct {
	MinSize    int
	MaxSize    int
	DefaultTTL time.Duration

	// Sets a callback that is invoked whenever the periodic
	// eviction has been run. Allows for "tacking on" extra
	// cleanup tasks outside of the ImageCache's jurisdiction
	// that are run on the same schedule.
	OnEvictTaskRan func()

	mu    sync.RWMutex
	cache map[string]CacheItem
}

// ErrNotFound is returned when a requested cache item does not exist.
var ErrNotFound = errors.New("item not found")

// Init initializes the cache and starts a background goroutine for periodic eviction.
// The goroutine stops when the provided context is cancelled.
func (i *ImageCache) Init(ctx context.Context, evictionInterval time.Duration) {
	i.cache = make(map[string]CacheItem)
	go i.periodicallyEvict(ctx, evictionInterval)
}

// SetWithTTL stores an image in the cache with a custom time-to-live duration.
// If the cache is at MaxSize, an item will be evicted using LRU strategy.
// Thread-safe. Holds writer lock for O(MaxSize) worst case.
func (i *ImageCache) SetWithTTL(key string, val image.Image, ttl time.Duration) {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now().Unix()
	if v, ok := i.cache[key]; ok {
		v.val = val
		v.ttl = ttl
		v.expiresAt = time.Now().Add(v.ttl).Unix()
		v.lastAccessed = now
		i.cache[key] = v // Update the map with modified struct
		return
	}
	if len(i.cache) == i.MaxSize {
		i.evictOne()
	}
	i.cache[key] = CacheItem{
		val:          val,
		ttl:          ttl,
		expiresAt:    time.Now().Add(ttl).Unix(),
		lastAccessed: now,
	}
}

// Set stores an image in the cache with the default TTL.
// See SetWithTTL for more details.
func (i *ImageCache) Set(key string, val image.Image) {
	i.SetWithTTL(key, val, i.DefaultTTL)
}

// Has returns true if the key exists in the cache, expired or not.
// Thread-safe.
func (i *ImageCache) Has(key string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	_, ok := i.cache[key]
	return ok
}

// Get retrieves an image from the cache and updates its last accessed time.
// Returns ErrNotFound if the key doesn't exist.
// Thread-safe.
func (i *ImageCache) Get(key string) (image.Image, error) {
	return i.GetResetTTL(key, false)
}

// GetResetTTL retrieves an image and optionally resets its expiration time.
// If resetTTL is true, the expiration is reset to now + original TTL.
// Returns ErrNotFound if the key doesn't exist.
// Thread-safe.
func (i *ImageCache) GetResetTTL(key string, resetTTL bool) (image.Image, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		if resetTTL {
			v.expiresAt = time.Now().Add(v.ttl).Unix()
		}
		i.cache[key] = v // Update the map with modified struct
		return v.val, nil
	}
	return nil, ErrNotFound
}

// GetExtendTTL retrieves an image and extends its TTL if it would expire sooner.
// The expiration time is extended to now + ttl only if the current expiration
// is earlier than that time.
// Returns ErrNotFound if the key doesn't exist.
// Thread-safe.
func (i *ImageCache) GetExtendTTL(key string, ttl time.Duration) (image.Image, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		newExpiry := time.Now().Add(ttl).Unix()
		if v.expiresAt < newExpiry {
			v.expiresAt = newExpiry
		}
		i.cache[key] = v // Update the map with modified struct
		return v.val, nil
	}
	return nil, ErrNotFound
}

// GetWithNewTTL retrieves an image and replaces its TTL with a new value.
// The expiration time is set to now + newTtl.
// Returns ErrNotFound if the key doesn't exist.
// Thread-safe.
func (i *ImageCache) GetWithNewTTL(key string, newTtl time.Duration) (image.Image, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if v, ok := i.cache[key]; ok {
		v.lastAccessed = time.Now().Unix()
		v.expiresAt = time.Now().Add(newTtl).Unix()
		v.ttl = newTtl
		i.cache[key] = v // Update the map with modified struct
		return v.val, nil
	}
	return nil, ErrNotFound
}

// Clear removes all items from the cache.
// Thread-safe.
func (i *ImageCache) Clear() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.cache = make(map[string]CacheItem)
}

// must be called when rwmutex is already acquired for writing
func (i *ImageCache) evictOne() {
	now := time.Now().Unix()
	var lruKey string
	lruTime := now + 1 // Initialize to future time so any item will be less
	var lruExpiredKey string
	lruExpiredTime := now + 1 // Initialize to future time
	hasExpired := false

	// Single pass through the cache to find both LRU expired and LRU items
	for k, v := range i.cache {
		if v.expiresAt < now {
			// This item is expired
			if v.lastAccessed < lruExpiredTime {
				lruExpiredTime = v.lastAccessed
				lruExpiredKey = k
				hasExpired = true
			}
		}
		// Track LRU regardless of expiration
		if v.lastAccessed < lruTime {
			lruTime = v.lastAccessed
			lruKey = k
		}
	}

	if hasExpired {
		// Prefer deleting LRU expired item
		delete(i.cache, lruExpiredKey)
	} else {
		// No expired items, delete LRU non-expired item
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
			if i.OnEvictTaskRan != nil {
				i.OnEvictTaskRan()
			}
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
