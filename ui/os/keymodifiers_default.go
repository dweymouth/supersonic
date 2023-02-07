//go:build !darwin

package os

import "fyne.io/fyne/v2"

var (
	ControlModifier = fyne.KeyModifierControl
	AltModifier     = fyne.KeyModifierAlt
)
