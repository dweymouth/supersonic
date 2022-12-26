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

func (l *LibraryManager) RecentlyAddedIter() AlbumIterator {
	return l.newBaseIter("newest")
}

func (l *LibraryManager) RecentlyPlayedIter() AlbumIterator {
	return l.newBaseIter("recent")
}

func (l *LibraryManager) StarredIter() AlbumIterator {
	return l.newBaseIter("starred")
}

func (l *LibraryManager) FrequentlyPlayedIter() AlbumIterator {
	return l.newBaseIter("frequent")
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

// TODO: figure out why the iterator sometimes returns an album twice

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
