//go:build !darwin

package backend

import (
	"errors"
)

func InitMPMediaHandler(playbackManager *PlaybackManager, artURLLookup func(trackID string) (string, error)) error {
	// MPMediaHandler only supports macOS.
	return errors.New("unsupported platform")
}
