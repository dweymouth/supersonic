//go:build !windows

package windows

import "fyne.io/fyne/v2"

func SendNotification(n *fyne.Notification, iconFilePath string) {
	fyne.LogError("windows.SendNotification should not be invoked on non-Windows platform", nil)
	fyne.CurrentApp().SendNotification(n)
}
