package backend

import (
	"log"
	"strconv"

	"github.com/bluele/gcache"
	subsonic "github.com/dweymouth/go-subsonic"
)

type AlbumIterator interface {
	Next() *subsonic.AlbumID3
	NextN(int, func(*subsonic.AlbumID3))
}

type LibraryManager struct {
	s          *ServerManager
	albumCache gcache.Cache
}

func NewLibraryManager(s *ServerManager) *LibraryManager {
	cache := gcache.New(250).LRU().Build()
	return &LibraryManager{
		s:          s,
		albumCache: cache,
	}
}

type AlbumSortOrder string

const (
	RecentlyAdded    AlbumSortOrder = "Recently Added"
	RecentlyPlayed   AlbumSortOrder = "Recently Played"
	FrequentlyPlayed AlbumSortOrder = "Frequently Played"
	TitleAZ          AlbumSortOrder = "Title (A-Z)"
	ArtistAZ         AlbumSortOrder = "Artist (A-Z)"
)

var (
	AlbumSortOrders []string = []string{
		string(RecentlyAdded),
		string(RecentlyPlayed),
		string(FrequentlyPlayed),
		string(TitleAZ),
		string(ArtistAZ),
	}
)

func (l *LibraryManager) AlbumsIter(sort AlbumSortOrder) AlbumIterator {
	switch sort {
	case RecentlyAdded:
		return l.newBaseIter("newest")
	case RecentlyPlayed:
		return l.newBaseIter("recent")
	case FrequentlyPlayed:
		return l.newBaseIter("frequent")
	case TitleAZ:
		return l.newBaseIter("alphabeticalByName")
	case ArtistAZ:
		return l.newBaseIter("alphabeticalByArtist")
	default:
		log.Printf("Undefined album sort order: %s", sort)
		return nil
	}
}

func (l *LibraryManager) StarredIter() AlbumIterator {
	return l.newBaseIter("starred")
}

func (l *LibraryManager) SearchIter(query string) AlbumIterator {
	return l.newSearchIter(query)
}

func (l *LibraryManager) CacheAlbum(a *subsonic.AlbumID3) {
	l.albumCache.Set(a.ID, a)
}

func (l *LibraryManager) GetAlbum(id string) (*subsonic.AlbumID3, error) {
	if l.albumCache.Has(id) {
		if a, err := l.albumCache.Get(id); err == nil {
			return a.(*subsonic.AlbumID3), nil
		}
	}
	a, err := l.s.Server.GetAlbum(id)
	if err != nil {
		return nil, err
	}
	l.albumCache.Set(a.ID, a)
	return a, nil
}

type baseIter struct {
	listType      string
	pos           int
	l             *LibraryManager
	s             *subsonic.Client
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	done          bool
}

func (l *LibraryManager) newBaseIter(listType string) *baseIter {
	return &baseIter{
		listType: listType,
		l:        l,
		s:        l.s.Server,
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

		r.l.CacheAlbum(a)
		return a
	}
	albums, err := r.s.GetAlbumList2(r.listType, map[string]string{"size": "20", "offset": strconv.Itoa(r.pos)})
	if err != nil {
		log.Println(err)
		albums = nil
	}
	if len(albums) == 0 {
		r.done = true
		return nil
	} else if len(albums) == 1 {
		r.l.CacheAlbum(albums[0])
		r.done = true
		return albums[0]
	}
	r.prefetched = albums
	r.prefetchedPos = 1

	r.l.CacheAlbum(r.prefetched[0])
	return r.prefetched[0]
}

func (r *baseIter) NextN(n int, cb func(*subsonic.AlbumID3)) {
	go func() {
		for i := 0; i < n; i++ {
			a := r.Next()
			cb(a)
			if a == nil {
				break
			}
		}
	}()
}

type searchIter struct {
	query         string
	artistOffset  int
	albumOffset   int
	songOffset    int
	l             *LibraryManager
	s             *subsonic.Client
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	albumIDset    map[string]bool
	done          bool
}

func (l *LibraryManager) newSearchIter(query string) *searchIter {
	return &searchIter{
		query:      query,
		l:          l,
		s:          l.s.Server,
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

		s.l.CacheAlbum(a)
		return a
	}

	return nil
}

func (s *searchIter) NextN(n int, cb func(*subsonic.AlbumID3)) {
	go func() {
		for i := 0; i < n; i++ {
			a := s.Next()
			cb(a)
			if a == nil {
				break
			}
		}
	}()
}

func (s *searchIter) addNewAlbums(al []*subsonic.AlbumID3) {
	for _, album := range al {
		if _, have := s.albumIDset[album.ID]; have {
			continue
		}
		s.prefetched = append(s.prefetched, album)
		s.albumIDset[album.ID] = true
	}
}
