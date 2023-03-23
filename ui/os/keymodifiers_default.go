//go:build !darwin

package os

import "fyne.io/fyne/v2"

const (
	ControlModifier = fyne.KeyModifierControl
	AltModifier     = fyne.KeyModifierAlt
)

var (
	BackShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierAlt, KeyName: fyne.KeyLeft},
	}
	ForwardShortcuts = []desktop.CustomShortcut{
		{Modifier: fyne.KeyModifierAlt, KeyName: fyne.KeyRight},
	}
)
