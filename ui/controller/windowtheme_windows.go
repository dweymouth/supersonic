//go:build windows

package controller

import (
	"runtime"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"golang.org/x/sys/windows/registry"
)

var (
	dwm    = syscall.NewLazyDLL("dwmapi.dll")
	setAtt = dwm.NewProc("DwmSetWindowAttribute")
)

func setWindowDarkTheme(hwnd uintptr, mode int) {
	if runtime.GOOS != "windows" {
		return
	}
	arg := 0
	switch mode {
	case 0: /*auto*/
		if isDark() {
			arg = 1
		}
	case 1: /*dark*/
		arg = 1
	}

	// copied from Fyne internals
	ret, _, err := setAtt.Call(uintptr(unsafe.Pointer(hwnd)), // window handle
		20,                            // DWMWA_USE_IMMERSIVE_DARK_MODE
		uintptr(unsafe.Pointer(&arg)), // on or off
		4)                             // sizeof(bool for windows))
	if ret != 0 && ret != 0x80070057 { // err is always non-nil, we check return value (except erroneous code)
		fyne.LogError("Failed to set dark mode", err)
	}
}

// copied from Fyne internals
func isDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil { // older version of Windows will not have this key
		return false
	}
	defer k.Close()

	useLight, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil { // older version of Windows will not have this value
		return false
	}

	return useLight == 0
}
