//go:build !windows

package windows

import "errors"

func InitializeTaskbarButtons(hwnd uintptr, callback func(TaskbarButton)) error {
	return errors.New("taskbar buttons unsupported")
}
