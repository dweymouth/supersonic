package backend

import (
	"log"

	"github.com/dweymouth/go-subsonic/subsonic"
)

func (l *LibraryManager) AllTracksIterator() TrackIterator {
	return &allTracksIterator{
		l:         l,
		albumIter: l.AlbumsIter(AlbumSortArtistAZ),
	}
}

func (l *LibraryManager) SearchTracksIterator(query string) TrackIterator {
	return &searchTracksIterator{
		searchIterBase: searchIterBase{
			s:     l.s.Server,
			query: query,
		},
		trackIDset: make(map[string]bool),
	}
}

type allTracksIterator struct {
	l           *LibraryManager
	albumIter   AlbumIterator
	curAlbum    *subsonic.AlbumID3
	curTrackIdx int
	done        bool
}

func (a *allTracksIterator) Next() *subsonic.Child {
	if a.done {
		return nil
	}

	// fetch next album
	if a.curAlbum == nil || a.curTrackIdx >= len(a.curAlbum.Song) {
		al := a.albumIter.Next()
		if al == nil {
			a.done = true
			return nil
		}
		al, err := a.l.s.Server.GetAlbum(al.ID)
		if err != nil {
			log.Printf("error fetching album: %s", err.Error())
		}
		if len(al.Song) == 0 {
			// in the unlikely case of an album with zero tracks,
			// just call recursively to move to next album
			return a.Next()
		}
		a.curAlbum = al
		a.curTrackIdx = 0
	}

	tr := a.curAlbum.Song[a.curTrackIdx]
	a.curTrackIdx += 1
	return tr
}

type searchTracksIterator struct {
	searchIterBase

	prefetched    []*subsonic.Child
	prefetchedPos int
	trackIDset    map[string]bool
	done          bool
}

func (s *searchTracksIterator) Next() *subsonic.Child {
	if s.done {
		return nil
	}

	// prefetch more search results from server
	if s.prefetched == nil {
		results := s.searchIterBase.fetchResults()
		if results == nil || len(results.Album)+len(results.Artist)+len(results.Song) == 0 {
			s.done = true
			s.trackIDset = nil
			return nil
		}

		// add results from songs search
		s.addNewTracks(results.Song)
		s.songOffset += len(results.Song)

		// add results from artists search
		for _, artist := range results.Artist {
			artist, err := s.s.GetArtist(artist.ID)
			if err != nil {
				log.Printf("error fetching artist: %s", err.Error())
			} else {
				s.addNewTracksFromAlbums(artist.Album)
			}
		}
		s.artistOffset += len(results.Artist)

		// add results from albums search
		s.addNewTracksFromAlbums(results.Album)
		s.albumOffset += len(results.Album)
	}

	// return from prefetched results
	if len(s.prefetched) > 0 {
		tr := s.prefetched[s.prefetchedPos]
		s.prefetchedPos++
		if s.prefetchedPos == len(s.prefetched) {
			s.prefetched = nil
			s.prefetchedPos = 0
		}

		return tr
	}

	return nil
}

func (s *searchTracksIterator) addNewTracks(tracks []*subsonic.Child) {
	for _, tr := range tracks {
		if _, have := s.trackIDset[tr.ID]; have {
			continue
		}
		s.prefetched = append(s.prefetched, tr)
		s.trackIDset[tr.ID] = true
	}
}

func (s *searchTracksIterator) addNewTracksFromAlbums(albums []*subsonic.AlbumID3) {
	for _, al := range albums {
		if album, err := s.s.GetAlbum(al.ID); err != nil {
			log.Printf("error fetching album: %s", err.Error())
		} else {
			s.addNewTracks(album.Song)
		}
	}
}
