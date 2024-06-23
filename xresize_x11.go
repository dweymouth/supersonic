//go:build linux && !wayland

package main

/*
#cgo LDFLAGS: -lX11
#include "xresize.h"
*/
import (
	"C"
)

func SendResizeToPID(pid, w, h int) {
	C.send_resize_to_pid(C.int(pid), C.int(w), C.int(h))
}
