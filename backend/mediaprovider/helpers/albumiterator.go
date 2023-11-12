package helpers

import (
	"log"
	"strings"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type AlbumFetchFn func(offset, limit int) ([]*mediaprovider.Album, error)

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

func NewRandomIter(deterministicFetcher, randomFetcher AlbumFetchFn, filter mediaprovider.AlbumFilter, prefetchCoverCB func(string)) *randomIter {
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
				if _, ok := r.albumIDSet[album.ID]; !ok && filterMatches(r.filter, album, false) {
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
					if filterMatches(r.filter, album, false) {
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

func filterMatches(f mediaprovider.AlbumFilter, album *mediaprovider.Album, ignoreGenre bool) bool {
	if album == nil {
		return false
	}
	if f.ExcludeFavorited && album.Favorite {
		return false
	}
	if f.ExcludeUnfavorited && !album.Favorite {
		return false
	}
	if y := album.Year; y < f.MinYear || (f.MaxYear > 0 && y > f.MaxYear) {
		return false
	}
	if ignoreGenre || len(f.Genres) == 0 {
		return true
	}
	return genresMatch(f.Genres, album.Genres)
}

func genresMatch(filterGenres, albumGenres []string) bool {
	for _, g1 := range filterGenres {
		for _, g2 := range albumGenres {
			if strings.EqualFold(g1, g2) {
				return true
			}
		}
	}
	return false
}
