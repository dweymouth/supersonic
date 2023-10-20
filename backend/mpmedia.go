package backend

import (
	"github.com/dweymouth/supersonic/player"
)

// MPMediaHandler is the handler for MacOS media controls and system events.
type MPMediaHandler struct {
	player          *player.Player
	playbackManager *PlaybackManager
	ArtURLLookup    func(trackID string) (string, error)
}
