//go:build darwin

package ui

/*
void installReopenDelegate();
*/
import "C"

import "fyne.io/fyne/v2"

var darwinAppDelegateReopenWindow fyne.Window

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
