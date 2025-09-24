//go:build !windows

package windows

import "errors"

type (
	SMTCPlaybackState int
	SMTCButton        int
)

const (
	// constants from smtc.h in github.com/supersonic-app/smtc-dll
	SMTCPlaybackStateStopped SMTCPlaybackState = 2
	SMTCPlaybackStatePlaying SMTCPlaybackState = 3
	SMTCPlaybackStatePaused  SMTCPlaybackState = 4

	SMTCButtonPlay     SMTCButton = 0
	SMTCButtonPause    SMTCButton = 1
	SMTCButtonStop     SMTCButton = 2
	SMTCButtonPrevious SMTCButton = 4
	SMTCButtonNext     SMTCButton = 5
)

type SMTC struct{}

var errSMTCUnsupported = errors.New("no support for SMTC on this platform")

func InitSMTCForWindow(hwnd uintptr) (*SMTC, error) {
	return nil, errSMTCUnsupported
}

func (s *SMTC) SetEnabled(enabled bool) error {
	return errSMTCUnsupported
}

func (s *SMTC) SetThumbnail(filepath string) error {
	return errSMTCUnsupported
}

func (s *SMTC) OnButtonPressed(func(SMTCButton)) {}

func (s *SMTC) OnSeek(f func(millis int)) {}

func (s *SMTC) Shutdown() {}

func (s *SMTC) UpdatePlaybackState(state SMTCPlaybackState) error {
	return errSMTCUnsupported
}

func (s *SMTC) UpdateMetadata(title, artist string) error {
	return errSMTCUnsupported
}

func (s *SMTC) UpdatePosition(positionMillis, durationMillis int) error {
	return errSMTCUnsupported
}
