//go:build windows || darwin

package dialogs

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	sqDialog "github.com/sqweek/dialog"
)

type NativeFileDialog struct {
	isSaveDlg    bool
	native       *sqDialog.FileBuilder
	fallback     *dialog.FileDialog
	openCallback func(fyne.URIReadCloser, error)
	saveCallback func(fyne.URIWriteCloser, error)
}

func NewNativeFileOpen(callback func(fyne.URIReadCloser, error), parent fyne.Window) *NativeFileDialog {
	n := &NativeFileDialog{
		openCallback: callback,
		native:       sqDialog.File(),
		fallback:     dialog.NewFileOpen(callback, parent),
	}
	return n
}

func NewNativeFileSave(callback func(fyne.URIWriteCloser, error), parent fyne.Window) *NativeFileDialog {
	n := &NativeFileDialog{
		isSaveDlg:    true,
		saveCallback: callback,
		native:       sqDialog.File(),
		fallback:     dialog.NewFileSave(callback, parent),
	}
	return n
}

func (n *NativeFileDialog) SetLocation(u fyne.ListableURI) {
	n.native.StartDir = u.Path()
	n.fallback.SetLocation(u)
}

func (n *NativeFileDialog) SetFilter(filter storage.FileFilter) {
	// MimeTypeFileFilter is not supported
	if ext, ok := filter.(*storage.ExtensionFileFilter); ok {
		exts := make([]string, 0, len(ext.Extensions))
		for _, e := range ext.Extensions {
			exts = append(exts, strings.TrimPrefix(e, "."))
		}
		n.native.Filters = []sqDialog.FileFilter{
			{Extensions: exts},
		}
	}
	n.fallback.SetFilter(filter)
}

func (n *NativeFileDialog) SetFileName(fileName string) {
	n.native.StartFile = fileName
	n.fallback.SetFileName(fileName)
}

func (n *NativeFileDialog) Show() {
	go func() {
		var choice string
		var err error
		if n.isSaveDlg {
			choice, err = n.native.Save()
		} else {
			choice, err = n.native.Load()
		}
		if err == nil || err == sqDialog.ErrCancelled {
			if n.isSaveDlg {
				if err == sqDialog.ErrCancelled {
					n.saveCallback(nil, nil)
				} else if writer, err := storage.Writer(storage.NewFileURI(choice)); err == nil {
					n.saveCallback(writer, nil)
				}
			} else {
				if err == sqDialog.ErrCancelled {
					n.openCallback(nil, nil)
				} else if reader, err := storage.Reader(storage.NewFileURI(choice)); err == nil {
					n.openCallback(reader, nil)
				}
			}
		} else {
			n.fallback.Show()
		}
	}()
}
