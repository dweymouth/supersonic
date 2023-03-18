package backend

import "github.com/dweymouth/go-subsonic/subsonic"

type allTracksIterator struct {
	albumIter   AlbumIterator
	curAlbum    *subsonic.AlbumID3
	curTrackIdx int
	done        bool
}

func (l *LibraryManager) AllTracksIterator() TrackIterator {
	return &allTracksIterator{
		albumIter: l.AlbumsIter(AlbumSortRecentlyAdded),
	}
}

func (a *allTracksIterator) Next() *subsonic.Child {
	if a.done {
		return nil
	}

	// fetch next album
	if a.curAlbum == nil || a.curTrackIdx >= len(a.curAlbum.Song) {
		a.curAlbum = a.albumIter.Next()
		if a.curAlbum == nil {
			a.done = true
			return nil
		}
		a.curTrackIdx = 0

		if len(a.curAlbum.Song) == 0 {
			// in the unlikely case of an album with zero tracks,
			// just call recursively to move to next album
			return a.Next()
		}
	}

	tr := a.curAlbum.Song[a.curTrackIdx]
	a.curTrackIdx += 1
	return tr
}
