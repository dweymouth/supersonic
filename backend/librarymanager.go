package backend

import (
	subsonic "github.com/dweymouth/go-subsonic/subsonic"
)

type AlbumIterator interface {
	Next() *subsonic.AlbumID3
}

type LibraryManager struct {
	PreCacheCoverFn func(string)

	s *ServerManager
}

func NewLibraryManager(s *ServerManager) *LibraryManager {
	return &LibraryManager{
		s: s,
	}
}

func (l *LibraryManager) GetUserOwnedPlaylists() ([]*subsonic.Playlist, error) {
	pl, err := l.s.Server.GetPlaylists(nil)
	userPl := make([]*subsonic.Playlist, 0)
	if err != nil {
		return nil, err
	}
	for _, p := range pl {
		if p.Owner == l.s.Server.User {
			userPl = append(userPl, p)
		}
	}
	return userPl, nil
}
