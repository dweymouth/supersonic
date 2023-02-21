package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type TappableIcon struct {
	widget.Icon

	OnTapped func()
}

func NewTappbaleIcon(res fyne.Resource) *TappableIcon {
	icon := &TappableIcon{}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)

	return icon
}

func (t *TappableIcon) Tapped(_ *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

func (t *TappableIcon) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}
