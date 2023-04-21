package backend

import (
	"log"

	"github.com/dweymouth/go-subsonic/subsonic"
)

type allTracksIterator struct {
	l           *LibraryManager
	albumIter   AlbumIterator
	curAlbum    *subsonic.AlbumID3
	curTrackIdx int
	done        bool
}

func (l *LibraryManager) AllTracksIterator() TrackIterator {
	return &allTracksIterator{
		l:         l,
		albumIter: l.AlbumsIter(AlbumSortArtistAZ),
	}
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
