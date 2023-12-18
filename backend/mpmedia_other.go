//go:build !darwin

package backend

import (
	"errors"

	"github.com/dweymouth/supersonic/player/mpv"
)

func InitMPMediaHandler(player *mpv.Player, playbackManager *PlaybackManager, artURLLookup func(trackID string) (string, error)) error {
	// MPMediaHandler only supports macOS.
	return errors.New("unsupported platform")
}
