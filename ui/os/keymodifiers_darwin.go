//go:build darwin

package os

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

const (
	ControlModifier = fyne.KeyModifierSuper
	AltModifier     = fyne.KeyModifierSuper
)

var (
	BackShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierSuper, KeyName: fyne.KeyLeft},
		{Modifier: fyne.KeyModifierSuper, KeyName: fyne.KeyLeftBracket},
	}
	ForwardShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierSuper, KeyName: fyne.KeyRight},
		{Modifier: fyne.KeyModifierSuper, KeyName: fyne.KeyRightBracket},
	}
)
