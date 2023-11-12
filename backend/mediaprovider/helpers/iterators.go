package helpers

import (
	"log"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

type Filter[T any] interface {
	IsNil() bool
	Matches(*T) bool
}

type baseIter[T any] struct {
	filter        Filter[T]
	prefetchCB    func(*T)
	serverPos     int
	fetcher       func(offset, limit int) ([]*T, error)
	prefetched    []*T
	prefetchedPos int
	done          bool
}

type AlbumFetchFn func(offset, limit int) ([]*mediaprovider.Album, error)

func NewAlbumIterator(fetchFn AlbumFetchFn, filter mediaprovider.AlbumFilter, cb func(string)) mediaprovider.AlbumIterator {
	return &baseIter[mediaprovider.Album]{
		prefetchCB: func(a *mediaprovider.Album) { cb(a.CoverArtID) },
		filter:     filter,
		fetcher:    fetchFn,
	}
}

type TrackFetchFn func(offset, limit int) ([]*mediaprovider.Track, error)

func NewTrackIterator(fetchFn TrackFetchFn, cb func(string)) mediaprovider.TrackIterator {
	return &baseIter[mediaprovider.Track]{
		prefetchCB: func(a *mediaprovider.Track) { cb(a.CoverArtID) },
		filter:     nilFilter[mediaprovider.Track]{},
		fetcher:    fetchFn,
	}
}

func (r *baseIter[T]) Next() *T {
	if r.done {
		return nil
	}
	if r.prefetched != nil && r.prefetchedPos < len(r.prefetched) {
		a := r.prefetched[r.prefetchedPos]
		r.prefetchedPos++
		return a
	}
	r.prefetched = nil
	for { // keep fetching until we are done or have mathcing results
		items, err := r.fetcher(r.serverPos, 20)
		if err != nil {
			log.Printf("error fetching items: %s", err.Error())
			items = nil
		}
		if len(items) == 0 {
			r.done = true
			return nil
		}
		r.serverPos += len(items)
		if !r.filter.IsNil() {
			items = sharedutil.FilterSlice(items, func(al *T) bool {
				return r.filter.Matches(al)
			})
		}
		r.prefetched = items
		if len(items) > 0 {
			break
		}
	}
	r.prefetchedPos = 1
	if r.prefetchCB != nil {
		for _, album := range r.prefetched {
			go r.prefetchCB(album)
		}
	}
	return r.prefetched[0]
}

type randomIter struct {
	filter        mediaprovider.AlbumFilter
	prefetchCB    func(coverArtID string)
	albumIDSet    map[string]bool
	prefetched    []*mediaprovider.Album
	prefetchedPos int
	// Random iter works in two phases - phase 1 by requesting random
	// albums from the server. Since the Subsonic API provides no way
	// of paginating a single random sort, we may get albums back twice.
	// We use albumIDSet to keep track of which albums have already been returned.
	// Once we start getting back too many already-returned albums,
	// switch to requesting more albums from a deterministic sort order.
	deterministicFetcher AlbumFetchFn
	ramdomFetcher        AlbumFetchFn
	phaseTwo             bool
	offset               int
	done                 bool
}

func NewRandomAlbumIter(deterministicFetcher, randomFetcher AlbumFetchFn, filter mediaprovider.AlbumFilter, prefetchCoverCB func(string)) *randomIter {
	return &randomIter{
		filter:               filter,
		prefetchCB:           prefetchCoverCB,
		deterministicFetcher: deterministicFetcher,
		ramdomFetcher:        randomFetcher,
		albumIDSet:           make(map[string]bool),
	}
}

func (r *randomIter) Next() *mediaprovider.Album {
	if r.done {
		return nil
	}

	// repeat fetch task until we have matching results
	// or we reach the end (handled via short circuit return)
	for len(r.prefetched) == 0 {
		if r.phaseTwo {
			// fetch albums from deterministic order
			albums, err := r.deterministicFetcher(r.offset, 25)
			if err != nil {
				log.Printf("error fetching albums: %s", err.Error())
				albums = nil
			}
			if len(albums) == 0 {
				r.done = true
				r.albumIDSet = nil
				return nil
			}
			r.offset += len(albums)
			for _, album := range albums {
				if _, ok := r.albumIDSet[album.ID]; !ok && r.filter.Matches(album) {
					r.prefetched = append(r.prefetched, album)
					if r.prefetchCB != nil {
						go r.prefetchCB(album.CoverArtID)
					}
					r.albumIDSet[album.ID] = true
				}
			}
		} else {
			albums, err := r.ramdomFetcher(0 /*offset - doesn't matter for random*/, 25)
			if err != nil {
				log.Println(err)
				r.done = true
				r.albumIDSet = nil
				return nil
			}
			var hitCount int
			for _, album := range albums {
				if _, ok := r.albumIDSet[album.ID]; !ok {
					// still need to keep track even if album is not matched
					// by the filter because we need to know when to move to phase two
					hitCount++
					r.albumIDSet[album.ID] = true
					if r.filter.Matches(album) {
						r.prefetched = append(r.prefetched, album)
						if r.prefetchCB != nil {
							go r.prefetchCB(album.CoverArtID)
						}
					}
				}
			}
			if successRatio := float64(hitCount) / float64(25); successRatio < 0.3 {
				r.phaseTwo = true
			}
		}
	}

	// return from prefetched results
	if len(r.prefetched) > 0 {
		a := r.prefetched[r.prefetchedPos]
		r.prefetchedPos++
		if r.prefetchedPos == len(r.prefetched) {
			r.prefetched = nil
			r.prefetchedPos = 0
		}

		return a
	}

	return nil
}

type nilFilter[T any] struct{}

func (n nilFilter[T]) IsNil() bool { return true }

func (n nilFilter[T]) Matches(*T) bool { return true }
