//go:build !windows && !darwin

package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

// On Ubuntu, using sqweek/dialog is crashing on launch, so just wrap Fyne dialog
// TODO: figure out why and enable native file dialogs on Linux.
type NativeFileDialog struct {
	fallback *dialog.FileDialog
}

func NewNativeFileOpen(callback func(fyne.URIReadCloser, error), parent fyne.Window) *NativeFileDialog {
	n := &NativeFileDialog{
		fallback: dialog.NewFileOpen(callback, parent),
	}
	return n
}

func NewNativeFileSave(callback func(fyne.URIWriteCloser, error), parent fyne.Window) *NativeFileDialog {
	n := &NativeFileDialog{
		fallback: dialog.NewFileSave(callback, parent),
	}
	return n
}

func (n *NativeFileDialog) SetLocation(u fyne.ListableURI) {
	n.fallback.SetLocation(u)
}

func (n *NativeFileDialog) SetFilter(filter storage.FileFilter) {
	n.fallback.SetFilter(filter)
}

func (n *NativeFileDialog) SetFileName(fileName string) {
	n.fallback.SetFileName(fileName)
}

func (n *NativeFileDialog) Show() {
	n.fallback.Show()
}
