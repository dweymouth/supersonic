//go:build windows

package backend

/*
#cgo CFLAGS: -I .
void btn_callback_cgo(int in);
void seek_callback_cgo(int in);
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type SMTCPlaybackState int
type SMTCButton int

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

type SMTC struct {
	dll *windows.DLL

	onButtonPressed func(SMTCButton)
	onSeek          func(int)
}

var smtcInstance *SMTC

func InitSMTCForWindow(hwnd uintptr) (*SMTC, error) {
	if maj, _, _ := windows.RtlGetNtVersionNumbers(); maj < 10 {
		return nil, errors.New("SMTC is not supported on Windows versions < 10")
	}

	dll, err := windows.LoadDLL("smtc.dll")
	if err != nil {
		return nil, err
	}

	proc, err := dll.FindProc("InitializeForWindow")
	if err != nil {
		return nil, err
	}

	hr, _, _ := proc.Call(hwnd, uintptr(unsafe.Pointer(C.btn_callback_cgo)), uintptr(unsafe.Pointer(C.seek_callback_cgo)))
	if hr < 0 {
		return nil, fmt.Errorf("InitializeForWindow failed with HRESULT=%d", hr)
	}

	smtcInstance = &SMTC{dll: dll}
	return smtcInstance, nil
}

func (s *SMTC) OnButtonPressed(f func(SMTCButton)) {
	s.onButtonPressed = f
}

func (s *SMTC) OnSeek(f func(millis int)) {
	s.onSeek = f
}

func (s *SMTC) Shutdown() {
	if s.dll == nil {
		return
	}
	proc, err := s.dll.FindProc("Destroy")
	if err == nil {
		proc.Call()
	}

	s.dll.Release()
	s.dll = nil
	smtcInstance = nil
}

func (s *SMTC) UpdatePlaybackState(state SMTCPlaybackState) error {
	if s.dll == nil {
		return errors.New("SMTC DLL not available")
	}

	proc, err := s.dll.FindProc("UpdatePlaybackState")
	if err != nil {
		return err
	}

	if hr, _, _ := proc.Call(uintptr(state)); hr < 0 {
		return fmt.Errorf("UpdatePlaybackState failed with HRESULT=%d", hr)
	}
	return nil
}

func (s *SMTC) UpdateMetadata(title, artist string) error {
	if s.dll == nil {
		return errors.New("SMTC DLL not available")
	}

	utfTitle, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return err
	}

	utfArtist, err := windows.UTF16PtrFromString(artist)
	if err != nil {
		return err
	}

	proc, err := s.dll.FindProc("UpdateMetadata")
	if err != nil {
		return err
	}

	hr, _, _ := proc.Call(uintptr(unsafe.Pointer(utfTitle)), uintptr(unsafe.Pointer(utfArtist)))
	if hr < 0 {
		return fmt.Errorf("UpdateMetadata failed with HRESULT=%d", hr)
	}
	return nil
}

func (s *SMTC) UpdatePosition(positionMillis, durationMillis int) error {
	if s.dll == nil {
		return errors.New("SMTC DLL not available")
	}

	proc, err := s.dll.FindProc("UpdatePosition")
	if err != nil {
		return err
	}

	hr, _, _ := proc.Call(uintptr(positionMillis), uintptr(durationMillis))
	if hr < 0 {
		return fmt.Errorf("UpdatePosition failed with HRESULT=%d", hr)
	}
	return nil
}

//export btnCallback
func btnCallback(in int) {
	if smtcInstance != nil && smtcInstance.onButtonPressed != nil {
		smtcInstance.onButtonPressed(SMTCButton(in))
	}
}

//export seekCallback
func seekCallback(millis int) {
	if smtcInstance != nil && smtcInstance.onSeek != nil {
		smtcInstance.onSeek(millis)
	}
}
