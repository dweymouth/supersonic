//go:build !windows

package backend

import "errors"

type SMTCPlaybackState int

const (
	SMTCPlaybackStateStopped SMTCPlaybackState = 0
	SMTCPlaybackStatePlaying SMTCPlaybackState = 1
	SMTCPlaybackStatePaused  SMTCPlaybackState = 2
)

type SMTC struct{}

var smtcUnsupportedErr = errors.New("SMTC is not supported on this platformo")

func InitSMTCForWindow(hwnd uintptr) (*SMTC, error) {
	return nil, smtcUnsupportedErr
}

func (s *SMTC) UpdatePlaybackState(state SMTCPlaybackState) error {
	return smtcUnsupportedErr
}

func (s *SMTC) UpdateMetadata(title, artist string) error {
	return smtcUnsupportedErr
}

func (s *SMTC) Shutdown() {
}
