package subsonic

import (
	subsonicCli "github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type SubsonicServer struct {
	subsonicCli.Client
}

func (s *SubsonicServer) Login(username, password string) error {
	s.User = username
	return s.Client.Authenticate(password)
}

func (s *SubsonicServer) MediaProvider() mediaprovider.MediaProvider {
	return SubsonicMediaProvider(&s.Client)
}
