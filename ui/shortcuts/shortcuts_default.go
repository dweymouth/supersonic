//go:build !darwin

package shortcuts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

const (
	ControlModifier = fyne.KeyModifierControl
	AltModifier     = fyne.KeyModifierAlt
)

var (
	QuitShortcut  = &desktop.CustomShortcut{Modifier: fyne.KeyModifierControl, KeyName: fyne.KeyQ}
	BackShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierAlt, KeyName: fyne.KeyLeft},
	}
	ForwardShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierAlt, KeyName: fyne.KeyRight},
	}
	SettingsShortcut *desktop.CustomShortcut = nil // TODO: is there a platform standard for Win/Linux?
)
