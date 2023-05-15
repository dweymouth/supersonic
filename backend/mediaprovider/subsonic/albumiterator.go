package subsonic

import (
	"log"
	"strconv"
	"strings"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

func filterMatches(f mediaprovider.AlbumFilter, album *subsonic.AlbumID3) bool {
	if album == nil {
		return false
	}
	if f.ExcludeFavorited && !album.Starred.IsZero() {
		return false
	}
	if f.ExcludeUnfavorited && album.Starred.IsZero() {
		return false
	}
	if y := album.Year; y < f.MinYear || (f.MaxYear > 0 && y > f.MaxYear) {
		return false
	}
	if len(f.Genres) == 0 {
		return true
	}
	for _, g := range f.Genres {
		if strings.EqualFold(g, album.Genre) {
			return true
		}
	}
	return false
}

func filterIsEmpty(f mediaprovider.AlbumFilter) bool {
	return !f.ExcludeFavorited && !f.ExcludeUnfavorited &&
		f.MinYear == 0 && f.MaxYear == 0 && len(f.Genres) == 0
}

func (s *subsonicMediaProvider) IterateAlbums(sortOrder, searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	if searchQuery != "" {
		return s.newSearchIter(searchQuery, filter)
	}
	if sortOrder == "" && len(filter.Genres) == 1 {
		return s.newBaseIter("byGenre", filter, map[string]string{"genre": filter.Genres[0]})
	}
	if sortOrder == "" && filter.ExcludeUnfavorited {
		return s.newBaseIter("starred", filter, make(map[string]string))
	}
	switch sortOrder {
	case AlbumSortRecentlyAdded:
		return s.newBaseIter("newest", filter, make(map[string]string))
	case AlbumSortRecentlyPlayed:
		return s.newBaseIter("recent", filter, make(map[string]string))
	case AlbumSortFrequentlyPlayed:
		return s.newBaseIter("frequent", filter, make(map[string]string))
	case AlbumSortRandom:
		return s.newRandomIter()
	case AlbumSortTitleAZ:
		return s.newBaseIter("alphabeticalByName", filter, make(map[string]string))
	case AlbumSortArtistAZ:
		return s.newBaseIter("alphabeticalByArtist", filter, make(map[string]string))
	case AlbumSortYearAscending:
		return s.newBaseIter("byYear", filter, map[string]string{"fromYear": "0", "toYear": "3000"})
	case AlbumSortYearDescending:
		return s.newBaseIter("byYear", filter, map[string]string{"fromYear": "3000", "toYear": "0"})
	default:
		log.Printf("Undefined album sort order: %s", sortOrder)
		return nil
	}
}

type baseIter struct {
	listType      string
	filter        mediaprovider.AlbumFilter
	serverPos     int
	s             *subsonic.Client
	opts          map[string]string
	prefetched    []*mediaprovider.Album
	prefetchedPos int
	done          bool
}

func (s *subsonicMediaProvider) newBaseIter(listType string, filter mediaprovider.AlbumFilter, opts map[string]string) *baseIter {
	return &baseIter{
		listType: listType,
		filter:   filter,
		s:        s.client,
		opts:     opts,
	}
}

func (r *baseIter) Next() *mediaprovider.Album {
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
		r.opts["offset"] = strconv.Itoa(r.serverPos)
		albums, err := r.s.GetAlbumList2(r.listType, r.opts)
		if err != nil {
			log.Printf("error fetching albums: %s", err.Error())
			albums = nil
		}
		if len(albums) == 0 {
			r.done = true
			return nil
		}
		r.serverPos += len(albums)
		albums = sharedutil.FilterSlice(albums, func(al *subsonic.AlbumID3) bool { return filterMatches(r.filter, al) })
		r.prefetched = sharedutil.MapSlice(albums, toAlbum)
		if len(albums) > 0 {
			break
		}
	}
	r.prefetchedPos = 1
	/*
		if r.l.PreCacheCoverFn != nil {
			for _, album := range r.prefetched {
				go r.l.PreCacheCoverFn(album.CoverArt)
			}
		}
	*/

	return r.prefetched[0]
}

type searchIter struct {
	searchIterBase

	filter        mediaprovider.AlbumFilter
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	albumIDset    map[string]bool
	done          bool
}

func (s *subsonicMediaProvider) newSearchIter(query string, filter mediaprovider.AlbumFilter) *searchIter {
	return &searchIter{
		searchIterBase: searchIterBase{
			query: query,
			s:     s.client,
		},
		filter:     filter,
		albumIDset: make(map[string]bool),
	}
}

func (s *searchIter) Next() *mediaprovider.Album {
	if s.done {
		return nil
	}

	// prefetch more search results from server
	if s.prefetched == nil {
		results := s.searchIterBase.fetchResults()
		if results == nil {
			s.done = true
			s.albumIDset = nil
			return nil
		}

		// add results from albums search
		s.addNewAlbums(results.Album)
		s.albumOffset += len(results.Album)

		// add results from artists search
		for _, artist := range results.Artist {
			artist, err := s.s.GetArtist(artist.ID)
			if err != nil || artist == nil {
				log.Printf("error fetching artist: %s", err.Error())
			} else {
				s.addNewAlbums(artist.Album)
			}
		}
		s.artistOffset += len(results.Artist)

		// add results from songs search
		for _, song := range results.Song {
			if song.AlbumID == "" {
				continue
			}
			album, err := s.s.GetAlbum(song.AlbumID)
			if err != nil || album == nil {
				log.Printf("error fetching album: %s", err.Error())
			} else {
				s.addNewAlbums([]*subsonic.AlbumID3{album})
			}
		}
		s.songOffset += len(results.Song)
	}

	// return from prefetched results
	if len(s.prefetched) > 0 {
		a := s.prefetched[s.prefetchedPos]
		s.prefetchedPos++
		if s.prefetchedPos == len(s.prefetched) {
			s.prefetched = nil
			s.prefetchedPos = 0
		}

		return toAlbum(a)
	}

	return nil
}

func (s *searchIter) addNewAlbums(al []*subsonic.AlbumID3) {
	for _, album := range al {
		if _, have := s.albumIDset[album.ID]; have {
			continue
		}
		if filterMatches(s.filter, album) {
			continue
		}
		s.prefetched = append(s.prefetched, album)
		/*
			if s.l.PreCacheCoverFn != nil {
				go s.l.PreCacheCoverFn(album.CoverArt)
			}
		*/
		s.albumIDset[album.ID] = true
	}
}

type randomIter struct {
	albumIDSet    map[string]bool
	s             *subsonic.Client
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	// Random iter works in two phases - phase 1 by requesting random
	// albums from the server. Since the Subsonic API provides no way
	// of paginating a single random sort, we may get albums back twice.
	// We use albumIDSet to keep track of which albums have already been returned.
	// Once we start getting back too many already-returned albums,
	// switch to requesting more albums from a deterministic sort order.
	phaseTwo bool
	offset   int
	done     bool
}

func (s *subsonicMediaProvider) newRandomIter() *randomIter {
	return &randomIter{
		s:          s.client,
		albumIDSet: make(map[string]bool),
	}
}

func (r *randomIter) Next() *mediaprovider.Album {
	if r.done {
		return nil
	}

	if r.prefetched == nil {
		if r.phaseTwo {
			for len(r.prefetched) == 0 {
				albums, err := r.s.GetAlbumList2("newest", map[string]string{"size": "20", "offset": strconv.Itoa(r.offset)})
				if err != nil {
					log.Println(err)
					albums = nil
				}
				if len(albums) == 0 {
					r.done = true
					r.albumIDSet = nil
					return nil
				}
				r.offset += len(albums)
				for _, album := range albums {
					if _, ok := r.albumIDSet[album.ID]; !ok {
						r.prefetched = append(r.prefetched, album)
						/*
							if r.l.PreCacheCoverFn != nil {
								go r.l.PreCacheCoverFn(album.CoverArt)
							}
						*/
						r.albumIDSet[album.ID] = true
					}
				}
			}
			r.prefetchedPos = 0
		} else {
			albums, err := r.s.GetAlbumList2("random", map[string]string{"size": "25"})
			if err != nil {
				log.Println(err)
				r.done = true
				r.albumIDSet = nil
				return nil
			}
			var hitCount int
			for _, album := range albums {
				if _, ok := r.albumIDSet[album.ID]; !ok {
					hitCount++
					r.prefetched = append(r.prefetched, album)
					/*
						if r.l.PreCacheCoverFn != nil {
							go r.l.PreCacheCoverFn(album.CoverArt)
						}
					*/
					r.albumIDSet[album.ID] = true
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

		return toAlbum(a)
	}

	return nil
}

/*
type BatchingIterator struct {
	iter AlbumIterator
}

func NewBatchingIterator(iter AlbumIterator) *BatchingIterator {
	return &BatchingIterator{iter}
}

func (b *BatchingIterator) NextN(n int) []*subsonic.AlbumID3 {
	results := make([]*subsonic.AlbumID3, 0, n)
	i := 0
	for i < n {
		album := b.iter.Next()
		if album == nil {
			break
		}
		results = append(results, album)
		i++
	}
	return results
}
*/
