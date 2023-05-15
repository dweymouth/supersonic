package backend

import (
	subsonic "github.com/dweymouth/go-subsonic/subsonic"
)

type AlbumIterator interface {
	Next() *subsonic.AlbumID3
}

type TrackIterator interface {
	Next() *subsonic.Child
}

type LibraryManager struct {
	PreCacheCoverFn func(coverID string)

	s *ServerManager
}

func NewLibraryManager(s *ServerManager) *LibraryManager {
	return &LibraryManager{
		s: s,
	}
}
