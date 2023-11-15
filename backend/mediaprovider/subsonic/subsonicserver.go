package subsonic

import (
	subsonicCli "github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type SubsonicServer struct {
	subsonicCli.Client
}

func (s *SubsonicServer) Login(username, password string) mediaprovider.LoginResponse {
	s.User = username
	err := s.Client.Authenticate(password)
	return mediaprovider.LoginResponse{
		Error:       err,
		IsAuthError: err == subsonicCli.ErrAuthenticationFailure,
	}
}

func (s *SubsonicServer) MediaProvider() mediaprovider.MediaProvider {
	return SubsonicMediaProvider(&s.Client)
}
