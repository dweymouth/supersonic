package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AuxControls struct {
	widget.BaseWidget

	VolumeControl *VolumeControl

	container *fyne.Container
}

func NewAuxControls() *AuxControls {
	a := &AuxControls{
		VolumeControl: NewVolumeControl(),
	}
	a.container = container.NewHBox(layout.NewSpacer(), a.VolumeControl)
	return a
}

func (a *AuxControls) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type volumeSlider struct {
	widget.Slider

	Width float32
}

func NewVolumeSlider(width float32) *volumeSlider {
	v := &volumeSlider{
		Slider: widget.Slider{
			Min:         0,
			Max:         100,
			Step:        1,
			Orientation: widget.Horizontal,
			Value:       100,
		},
		Width: width,
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *volumeSlider) MinSize() fyne.Size {
	h := v.Slider.MinSize().Height
	return fyne.NewSize(v.Width, h)
}

type tappableIcon struct {
	widget.Icon

	OnTapped func()
}

func newTappableIcon(res fyne.Resource) *tappableIcon {
	icon := &tappableIcon{}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)

	return icon
}

func (t *tappableIcon) Tapped(_ *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *tappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

func (t *tappableIcon) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

type VolumeControl struct {
	widget.BaseWidget

	icon   *tappableIcon
	slider *volumeSlider

	OnVolumeChanged func(int)

	muted   bool
	lastVol int

	container *fyne.Container
}

func NewVolumeControl() *VolumeControl {
	v := &VolumeControl{}
	v.ExtendBaseWidget(v)
	v.icon = newTappableIcon(theme.VolumeUpIcon())
	v.icon.OnTapped = v.toggleMute
	v.slider = NewVolumeSlider(100)
	v.lastVol = 100
	v.slider.Step = 1
	v.slider.Orientation = widget.Horizontal
	v.slider.Value = 100
	v.slider.OnChanged = v.onChanged
	v.container = container.NewHBox(v.icon, v.slider)
	return v
}

func (v *VolumeControl) onChanged(volume float64) {
	vol := int(volume)
	v.lastVol = vol
	v.muted = false
	v.updateIconForVolume(vol)
	if v.OnVolumeChanged != nil {
		v.OnVolumeChanged(vol)
	}
}

func (v *VolumeControl) toggleMute() {
	if !v.muted {
		v.muted = true
		v.lastVol = int(v.slider.Value)
		v.SetVolume(0)
	} else {
		v.muted = false
		v.SetVolume(v.lastVol)
	}
}

func (v *VolumeControl) CreateRenderer() fyne.WidgetRenderer {
	v.ExtendBaseWidget(v)
	return widget.NewSimpleRenderer(v.container)
}

func (v *VolumeControl) SetVolume(vol int) {
	v.slider.Value = float64(vol)
	v.slider.Refresh()
	v.updateIconForVolume(vol)
	if v.OnVolumeChanged != nil {
		v.OnVolumeChanged(vol)
	}
}

func (v *VolumeControl) updateIconForVolume(vol int) {
	if vol <= 0 {
		v.icon.Resource = theme.VolumeMuteIcon()
	} else if vol < 50 {
		v.icon.Resource = theme.VolumeDownIcon()
	} else {
		v.icon.Resource = theme.VolumeUpIcon()
	}
	v.icon.Refresh()
}
