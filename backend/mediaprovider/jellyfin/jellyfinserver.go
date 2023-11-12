package jellyfin

import (
	jellyfinCli "github.com/dweymouth/go-jellyfin"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type JellyfinServer struct {
	jellyfinCli.Client
}

func (j *JellyfinServer) MediaProvider() mediaprovider.MediaProvider {
	return newJellyfinMediaProvider(&j.Client)
}
