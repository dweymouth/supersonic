//go:build !windows

package windows

import (
	"errors"
	"image"
)

var err = errors.New("taskbar buttons unsupported")

func InitializeTaskbarButtons(hwnd uintptr, callback func(TaskbarButton)) error {
	return err
}

func SetTaskbarButtonToolTips(prev, next, play, pause string) error {
	return err
}

func InitializeTaskbarIcons(prev, next, play, pause image.Image) error {
	return err
}

func SetTaskbarButtonIsPlaying(isPlaying bool) error {
	return err
}
