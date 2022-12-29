package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

type VolumeControl struct {
	widget.BaseWidget

	icon   *widget.Icon
	slider *volumeSlider

	OnVolumeChanged func(int)

	container *fyne.Container
}

func NewVolumeControl() *VolumeControl {
	v := &VolumeControl{}
	v.ExtendBaseWidget(v)
	v.icon = widget.NewIcon(theme.VolumeUpIcon())
	v.slider = NewVolumeSlider(100)
	v.slider.Step = 1
	v.slider.Orientation = widget.Horizontal
	v.slider.Value = 100
	v.slider.OnChanged = v.onChanged
	v.container = container.NewHBox(v.icon, v.slider)
	return v
}

func (v *VolumeControl) onChanged(volume float64) {
	vol := int(volume)
	if vol <= 0 {
		vol = 0
		v.icon.Resource = theme.VolumeMuteIcon()
	} else if vol < 50 {
		v.icon.Resource = theme.VolumeDownIcon()
	} else {
		if vol > 100 {
			vol = 100
		}
		v.icon.Resource = theme.VolumeUpIcon()
	}
	v.icon.Refresh()
	if v.OnVolumeChanged != nil {
		v.OnVolumeChanged(vol)
	}
}

func (v *VolumeControl) CreateRenderer() fyne.WidgetRenderer {
	v.ExtendBaseWidget(v)
	return widget.NewSimpleRenderer(v.container)
}

func (v *VolumeControl) SetVolume(vol int) {
	v.slider.Value = float64(vol)
}
