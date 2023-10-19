//go:build !darwin

package backend

import "github.com/dweymouth/supersonic/player"

func NewMPMediaHandler(player *player.Player, playbackManager *PlaybackManager) *MPMediaHandler {
	// MPMediaHandler only supports macOS.
	return nil
}
