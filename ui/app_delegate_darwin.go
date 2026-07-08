//go:build darwin

package ui

/*
void installReopenDelegate();
*/
import "C"

import "fyne.io/fyne/v2"

var darwinAppDelegateReopenWindow fyne.Window

// darwinQuitting is set to true when applicationShouldTerminate: fires so
// the close intercept knows to actually close instead of hide.
var darwinQuitting bool

func isRealQuit() bool {
	return darwinQuitting
}

func installReopenHandler(w fyne.Window) {
	darwinAppDelegateReopenWindow = w
	C.installReopenDelegate()
}

//export appReopened
func appReopened() {
	if darwinAppDelegateReopenWindow == nil {
		return
	}
	go func() {
		fyne.Do(darwinAppDelegateReopenWindow.Show)
	}()
}

//export appShouldTerminate
func appShouldTerminate() {
	darwinQuitting = true
}
