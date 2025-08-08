//go:build windows

package windows

/*
#cgo LDFLAGS: -lole32
#include "taskbar_buttons.h"

extern void goButtonClicked(int);
*/
import "C"
import (
	"unsafe"
)

var gTaskbarButtonCallback func(TaskbarButton)

func InitializeTaskbarButtons(hwnd uintptr, callback func(TaskbarButton)) error {
	gTaskbarButtonCallback = callback
	C.initialize_taskbar_buttons(unsafe.Pointer(hwnd), C.ThumbnailCallback(C.goButtonClicked))
	return nil
}

//export goButtonClicked
func goButtonClicked(buttonID C.int) {
	if gTaskbarButtonCallback != nil {
		gTaskbarButtonCallback(TaskbarButton(buttonID))
	}
}
