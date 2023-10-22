//go:build !darwin

package backend

import (
	"errors"

	"github.com/dweymouth/supersonic/player"
)

func InitMPMediaHandler(player *player.Player, playbackManager *PlaybackManager, artURLLookup func(trackID string) (string, error)) error {
	// MPMediaHandler only supports macOS.
	return errors.New("unsupported platform")
}
