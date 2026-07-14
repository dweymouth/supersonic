//go:build darwin

package controller

/*
void setWindowDarkMode(void* windowPtr, int useDarkMode);
*/
import "C"
import "unsafe"

// setWindowDarkTheme takes an unsafe.Pointer referencing the macOS NSWindow
// and explicitly applies or clears the dark mode appearance.
// Mode arg: 0 = auto, 1 = dark, 2 = light
func setWindowDarkTheme(nsWindowPtr uintptr, mode int) {
	C.setWindowDarkMode(unsafe.Pointer(nsWindowPtr), C.int(mode))
}
