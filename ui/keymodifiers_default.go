//go:build !darwin

package ui

import "fyne.io/fyne/v2"

var (
	ControlModifier = fyne.KeyModifierControl
	AltModifier     = fyne.KeyModifierAlt
)
