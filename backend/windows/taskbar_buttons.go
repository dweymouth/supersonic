//go:build windows

package windows

/*
#cgo LDFLAGS: -lole32
#include <stdlib.h>
#include "taskbar_buttons.h"

extern void goButtonClicked(int);
*/
import "C"

import (
	"errors"
	"image"
	"unsafe"
)

var gTaskbarButtonCallback func(TaskbarButton)

// InitializeTaskbarIcons supplies the image icons that will be used for the taskbar buttons.
// They must all have the same pixel dimensions.
// This should be called before InitializeTaskbarButtons or the buttons will not have icons.
func InitializeTaskbarIcons(prev, next, play, pause image.Image) error {
	pB := imageToBGRA(prev)
	nB := imageToBGRA(next)
	plB := imageToBGRA(play)
	paB := imageToBGRA(pause)
	bnds := prev.Bounds()

	C.initialize_taskbar_icons(unsafe.Pointer(&pB[0]), unsafe.Pointer(&nB[0]), unsafe.Pointer(&plB[0]), unsafe.Pointer(&paB[0]), C.int(bnds.Dx()), C.int(bnds.Dy()))
	return nil
}

// SetTaskbarButtonToolTips should be called before InitializeTaskbarButtons to
// set the tool tips that will be used for the buttons
func SetTaskbarButtonToolTips(prev, next, play, pause string) error {
	cPlay := C.CString(play)
	defer C.free(unsafe.Pointer(cPlay))
	cPause := C.CString(pause)
	defer C.free(unsafe.Pointer(cPause))
	cPrev := C.CString(prev)
	defer C.free(unsafe.Pointer(cPrev))
	cNext := C.CString(next)
	defer C.free(unsafe.Pointer(cNext))
	C.set_tooltips_utf8(cPrev, cNext, cPlay, cPause)

	return nil
}

func InitializeTaskbarButtons(hwnd uintptr, callback func(TaskbarButton)) error {
	gTaskbarButtonCallback = callback
	C.initialize_taskbar_buttons(unsafe.Pointer(hwnd), C.ThumbnailCallback(C.goButtonClicked))
	return nil
}

func SetTaskbarButtonIsPlaying(isPlaying bool) error {
	i := C.int(0)
	if isPlaying {
		i = C.int(1)
	}
	if ret := int(C.set_is_playing(i)); ret == 0 {
		return nil
	}
	return errors.New("failed to set taskbar button is playing state")
}

//export goButtonClicked
func goButtonClicked(buttonID C.int) {
	if gTaskbarButtonCallback != nil {
		gTaskbarButtonCallback(TaskbarButton(buttonID))
	}
}

// imageToBGRA converts an image.Image to a BGRA byte slice.
// For *image.RGBA, it un-premultiplies the colors to straight alpha.
func imageToBGRA(img image.Image) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	bgra := make([]byte, w*h*4)

	i := 0
	switch src := img.(type) {
	case *image.NRGBA:
		// Straight alpha, just swap channels
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			off := src.PixOffset(bounds.Min.X, y)
			for x := 0; x < w; x++ {
				r := src.Pix[off+0]
				g := src.Pix[off+1]
				b := src.Pix[off+2]
				a := src.Pix[off+3]
				bgra[i+0] = b
				bgra[i+1] = g
				bgra[i+2] = r
				bgra[i+3] = a
				off += 4
				i += 4
			}
		}

	case *image.RGBA:
		// Premultiplied alpha, un-premultiply to straight alpha
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			off := src.PixOffset(bounds.Min.X, y)
			for x := 0; x < w; x++ {
				r := src.Pix[off+0]
				g := src.Pix[off+1]
				b := src.Pix[off+2]
				a := src.Pix[off+3]

				if a != 0 {
					// Convert from premultiplied to straight alpha
					r = uint8((uint16(r) * 0xFF) / uint16(a))
					g = uint8((uint16(g) * 0xFF) / uint16(a))
					b = uint8((uint16(b) * 0xFF) / uint16(a))
				}

				bgra[i+0] = b
				bgra[i+1] = g
				bgra[i+2] = r
				bgra[i+3] = a
				off += 4
				i += 4
			}
		}

	default:
		// Fallback: handle any other image.Image type via At()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r16, g16, b16, a16 := img.At(x, y).RGBA()
				// Convert from 16-bit [0, 65535] to 8-bit [0, 255]
				r := uint8(r16 >> 8)
				g := uint8(g16 >> 8)
				b := uint8(b16 >> 8)
				a := uint8(a16 >> 8)
				bgra[i+0] = b
				bgra[i+1] = g
				bgra[i+2] = r
				bgra[i+3] = a
				i += 4
			}
		}
	}

	return bgra
}
