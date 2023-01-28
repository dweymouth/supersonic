package backend

import (
	"log"
	"strconv"

	"github.com/bluele/gcache"
	subsonic "github.com/dweymouth/go-subsonic"
)

type AlbumIterator interface {
	Next() *subsonic.AlbumID3
}

type LibraryManager struct {
	PreCacheCoverFn func(string)

	s                *ServerManager
	albumDetailCache gcache.Cache
}

func NewLibraryManager(s *ServerManager) *LibraryManager {
	cache := gcache.New(250).LRU().Build()
	return &LibraryManager{
		s:                s,
		albumDetailCache: cache,
	}
}

type AlbumSortOrder string

const (
	AlbumSortRecentlyAdded    AlbumSortOrder = "Recently Added"
	AlbumSortRecentlyPlayed   AlbumSortOrder = "Recently Played"
	AlbumSortFrequentlyPlayed AlbumSortOrder = "Frequently Played"
	AlbumSortRandom           AlbumSortOrder = "Random"
	AlbumSortTitleAZ          AlbumSortOrder = "Title (A-Z)"
	AlbumSortArtistAZ         AlbumSortOrder = "Artist (A-Z)"
)

var (
	AlbumSortOrders []string = []string{
		string(AlbumSortRecentlyAdded),
		string(AlbumSortRecentlyPlayed),
		string(AlbumSortFrequentlyPlayed),
		string(AlbumSortRandom),
		string(AlbumSortTitleAZ),
		string(AlbumSortArtistAZ),
	}
)

func (l *LibraryManager) AlbumsIter(sort AlbumSortOrder) AlbumIterator {
	switch sort {
	case AlbumSortRecentlyAdded:
		return l.newBaseIter("newest", make(map[string]string))
	case AlbumSortRecentlyPlayed:
		return l.newBaseIter("recent", make(map[string]string))
	case AlbumSortFrequentlyPlayed:
		return l.newBaseIter("frequent", make(map[string]string))
	case AlbumSortRandom:
		return l.newRandomIter()
	case AlbumSortTitleAZ:
		return l.newBaseIter("alphabeticalByName", make(map[string]string))
	case AlbumSortArtistAZ:
		return l.newBaseIter("alphabeticalByArtist", make(map[string]string))
	default:
		log.Printf("Undefined album sort order: %s", sort)
		return nil
	}
}

func (l *LibraryManager) StarredIter() AlbumIterator {
	return l.newBaseIter("starred", make(map[string]string))
}

func (l *LibraryManager) GenreIter(genre string) AlbumIterator {
	return l.newBaseIter("byGenre", map[string]string{"genre": genre})
}

func (l *LibraryManager) SearchIter(query string) AlbumIterator {
	return l.newSearchIter(query, func(*subsonic.AlbumID3) bool { return true })
}

func (l *LibraryManager) SearchIterWithFilter(query string, filter func(*subsonic.AlbumID3) bool) AlbumIterator {
	return l.newSearchIter(query, filter)
}

func (l *LibraryManager) CacheAlbum(a *subsonic.AlbumID3) {
	l.albumDetailCache.Set(a.ID, a)
}

func (l *LibraryManager) GetAlbum(id string) (*subsonic.AlbumID3, error) {
	if l.albumDetailCache.Has(id) {
		if a, err := l.albumDetailCache.Get(id); err == nil {
			return a.(*subsonic.AlbumID3), nil
		}
	}
	a, err := l.s.Server.GetAlbum(id)
	if err != nil {
		return nil, err
	}
	l.albumDetailCache.Set(a.ID, a)
	return a, nil
}

type baseIter struct {
	listType      string
	pos           int
	l             *LibraryManager
	s             *subsonic.Client
	opts          map[string]string
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	done          bool
}

func (l *LibraryManager) newBaseIter(listType string, opts map[string]string) *baseIter {
	return &baseIter{
		listType: listType,
		l:        l,
		s:        l.s.Server,
		opts:     opts,
	}
}

func (r *baseIter) Next() *subsonic.AlbumID3 {
	if r.done {
		return nil
	}
	if r.prefetched != nil {
		a := r.prefetched[r.prefetchedPos]
		r.prefetchedPos++
		if r.prefetchedPos == len(r.prefetched) {
			r.prefetched = nil
			r.prefetchedPos = 0
			r.pos++
		}
		r.pos++

		return a
	}
	r.opts["offset"] = strconv.Itoa(r.pos)
	albums, err := r.s.GetAlbumList2(r.listType, r.opts)
	if err != nil {
		log.Println(err)
		albums = nil
	}
	if len(albums) == 0 {
		r.done = true
		return nil
	} else if len(albums) == 1 {
		r.done = true
		return albums[0]
	}
	r.prefetched = albums
	r.prefetchedPos = 1
	if r.l.PreCacheCoverFn != nil {
		for _, album := range albums {
			go r.l.PreCacheCoverFn(album.ID)
		}
	}

	return r.prefetched[0]
}

type searchIter struct {
	query         string
	artistOffset  int
	albumOffset   int
	songOffset    int
	l             *LibraryManager
	s             *subsonic.Client
	filter        func(*subsonic.AlbumID3) bool
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	albumIDset    map[string]bool
	done          bool
}

func (l *LibraryManager) newSearchIter(query string, filter func(*subsonic.AlbumID3) bool) *searchIter {
	return &searchIter{
		query:      query,
		l:          l,
		s:          l.s.Server,
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
		searchOpts := map[string]string{
			"artistOffset": strconv.Itoa(s.artistOffset),
			"albumOffset":  strconv.Itoa(s.albumOffset),
			"songOffset":   strconv.Itoa(s.songOffset),
		}
		results, err := s.s.Search3(s.query, searchOpts)
		if err != nil {
			log.Println(err)
			results = nil
		}
		if results == nil || len(results.Album)+len(results.Artist)+len(results.Song) == 0 {
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
			if err != nil {
				log.Printf("error fetching artist: %s", err.Error())
			}
			s.addNewAlbums(artist.Album)
		}
		s.artistOffset += len(results.Artist)

		// add results from songs search
		for _, song := range results.Song {
			album, err := s.s.GetAlbum(song.Parent)
			if err != nil {
				log.Printf("error fetching album: %s", err.Error())
			}
			s.addNewAlbums([]*subsonic.AlbumID3{album})
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
		if !s.filter(album) {
			continue
		}
		s.prefetched = append(s.prefetched, album)
		if s.l.PreCacheCoverFn != nil {
			go s.l.PreCacheCoverFn(album.ID)
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
							go r.l.PreCacheCoverFn(album.ID)
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
						go r.l.PreCacheCoverFn(album.ID)
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
