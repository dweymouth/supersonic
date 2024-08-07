package subsonic

import (
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	subsonicCli "github.com/supersonic-app/go-subsonic/subsonic"
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
