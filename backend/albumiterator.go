package backend

import (
	"log"
	"strconv"
	"strings"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/sharedutil"
)

type AlbumSortOrder string

const (
	AlbumSortRecentlyAdded    AlbumSortOrder = "Recently Added"
	AlbumSortRecentlyPlayed   AlbumSortOrder = "Recently Played"
	AlbumSortFrequentlyPlayed AlbumSortOrder = "Frequently Played"
	AlbumSortRandom           AlbumSortOrder = "Random"
	AlbumSortTitleAZ          AlbumSortOrder = "Title (A-Z)"
	AlbumSortArtistAZ         AlbumSortOrder = "Artist (A-Z)"
	AlbumSortYearAscending    AlbumSortOrder = "Year (ascending)"
	AlbumSortYearDescending   AlbumSortOrder = "Year (descending)"
)

var (
	AlbumSortOrders []string = []string{
		string(AlbumSortRecentlyAdded),
		string(AlbumSortRecentlyPlayed),
		string(AlbumSortFrequentlyPlayed),
		string(AlbumSortRandom),
		string(AlbumSortTitleAZ),
		string(AlbumSortArtistAZ),
		string(AlbumSortYearAscending),
		string(AlbumSortYearDescending),
	}
)

type AlbumFilter struct {
	MinYear int
	MaxYear int      // 0 == unset/match any
	Genres  []string // len(0) == unset/match any

	ExcludeFavorited   bool // mut. exc. with ExcludeUnfavorited
	ExcludeUnfavorited bool // mut. exc. with ExcludeFavorited
}

func (f *AlbumFilter) Matches(album *subsonic.AlbumID3) bool {
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

func (f *AlbumFilter) IsEmpty() bool {
	return !f.ExcludeFavorited && !f.ExcludeUnfavorited &&
		f.MinYear == 0 && f.MaxYear == 0 && len(f.Genres) == 0
}

func (l *LibraryManager) AlbumsIter(sort AlbumSortOrder, filter AlbumFilter) AlbumIterator {
	switch sort {
	case AlbumSortRecentlyAdded:
		return l.newBaseIter("newest", filter, make(map[string]string))
	case AlbumSortRecentlyPlayed:
		return l.newBaseIter("recent", filter, make(map[string]string))
	case AlbumSortFrequentlyPlayed:
		return l.newBaseIter("frequent", filter, make(map[string]string))
	case AlbumSortRandom:
		return l.newRandomIter()
	case AlbumSortTitleAZ:
		return l.newBaseIter("alphabeticalByName", filter, make(map[string]string))
	case AlbumSortArtistAZ:
		return l.newBaseIter("alphabeticalByArtist", filter, make(map[string]string))
	case AlbumSortYearAscending:
		return l.newBaseIter("byYear", filter, map[string]string{"fromYear": "0", "toYear": "3000"})
	case AlbumSortYearDescending:
		return l.newBaseIter("byYear", filter, map[string]string{"fromYear": "3000", "toYear": "0"})
	default:
		log.Printf("Undefined album sort order: %s", sort)
		return nil
	}
}

func (l *LibraryManager) StarredIter(filter AlbumFilter) AlbumIterator {
	return l.newBaseIter("starred", filter, make(map[string]string))
}

func (l *LibraryManager) GenreIter(genre string, filter AlbumFilter) AlbumIterator {
	return l.newBaseIter("byGenre", filter, map[string]string{"genre": genre})
}

func (l *LibraryManager) SearchIter(query string) AlbumIterator {
	return l.newSearchIter(query, AlbumFilter{})
}

func (l *LibraryManager) SearchIterWithFilter(query string, filter AlbumFilter) AlbumIterator {
	return l.newSearchIter(query, filter)
}

func (l *LibraryManager) GetAlbum(id string) (*subsonic.AlbumID3, error) {
	a, err := l.s.Server.GetAlbum(id)
	if err != nil {
		return nil, err
	}
	return a, nil
}

type baseIter struct {
	listType      string
	filter        AlbumFilter
	serverPos     int
	l             *LibraryManager
	s             *subsonic.Client
	opts          map[string]string
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	done          bool
}

func (l *LibraryManager) newBaseIter(listType string, filter AlbumFilter, opts map[string]string) *baseIter {
	return &baseIter{
		listType: listType,
		filter:   filter,
		l:        l,
		s:        l.s.Server,
		opts:     opts,
	}
}

func (r *baseIter) Next() *subsonic.AlbumID3 {
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
		albums = sharedutil.FilterSlice(albums, r.filter.Matches)
		r.prefetched = albums
		if len(albums) > 0 {
			break
		}
	}
	r.prefetchedPos = 1
	if r.l.PreCacheCoverFn != nil {
		for _, album := range r.prefetched {
			go r.l.PreCacheCoverFn(album.CoverArt)
		}
	}

	return r.prefetched[0]
}

type searchIter struct {
	searchIterBase

	l             *LibraryManager
	filter        AlbumFilter
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	albumIDset    map[string]bool
	done          bool
}

func (l *LibraryManager) newSearchIter(query string, filter AlbumFilter) *searchIter {
	return &searchIter{
		searchIterBase: searchIterBase{
			query: query,
			s:     l.s.Server,
		},
		l:          l,
		filter:     filter,
		albumIDset: make(map[string]bool),
	}
}

func (s *searchIter) Next() *subsonic.AlbumID3 {
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

		return a
	}

	return nil
}

func (s *searchIter) addNewAlbums(al []*subsonic.AlbumID3) {
	for _, album := range al {
		if _, have := s.albumIDset[album.ID]; have {
			continue
		}
		if !s.filter.Matches(album) {
			continue
		}
		s.prefetched = append(s.prefetched, album)
		if s.l.PreCacheCoverFn != nil {
			go s.l.PreCacheCoverFn(album.CoverArt)
		}
		s.albumIDset[album.ID] = true
	}
}

type randomIter struct {
	albumIDSet    map[string]bool
	l             *LibraryManager
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

func (l *LibraryManager) newRandomIter() *randomIter {
	return &randomIter{
		l:          l,
		s:          l.s.Server,
		albumIDSet: make(map[string]bool),
	}
}

func (r *randomIter) Next() *subsonic.AlbumID3 {
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
						if r.l.PreCacheCoverFn != nil {
							go r.l.PreCacheCoverFn(album.CoverArt)
						}
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
					if r.l.PreCacheCoverFn != nil {
						go r.l.PreCacheCoverFn(album.CoverArt)
					}
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

		return a
	}

	return nil
}

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
