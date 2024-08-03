package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dweymouth/supersonic/backend"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

// The "aux" controls for playback, positioned to the right
// of the BottomPanel. Currently only volume control.
type AuxControls struct {
	widget.BaseWidget

	VolumeControl *VolumeControl
	loop          *IconButton
	showQueue     *IconButton

	container *fyne.Container
}

func NewAuxControls(initialVolume int) *AuxControls {
	a := &AuxControls{
		VolumeControl: NewVolumeControl(initialVolume),
		loop:          NewIconButton(myTheme.RepeatIcon, nil),
		showQueue:     NewIconButton(myTheme.PlayQueueIcon, nil),
	}
	a.loop.IconSize = IconButtonSizeSmaller
	a.loop.SetToolTip(lang.L("Shuffle"))
	a.showQueue.IconSize = IconButtonSizeSmaller
	a.showQueue.SetToolTip(lang.L("Show play queue"))
	a.container = container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			layout.NewSpacer(),
			a.VolumeControl,
			container.New(
				layout.NewCustomPaddedHBoxLayout(theme.Padding()*1.5),
				layout.NewSpacer(), a.loop, a.showQueue, util.NewHSpace(5)),
			layout.NewSpacer(),
		),
	)
	return a
}

func (a *AuxControls) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

func (a *AuxControls) OnChangeLoopMode(f func()) {
	a.loop.OnTapped = f
}

func (a *AuxControls) SetLoopMode(mode backend.LoopMode) {
	switch mode {
	case backend.LoopAll:
		a.loop.Highlighted = true
		a.loop.SetIcon(myTheme.RepeatIcon)
	case backend.LoopOne:
		a.loop.Highlighted = true
		a.loop.SetIcon(myTheme.RepeatOneIcon)
	case backend.LoopNone:
		a.loop.Highlighted = false
		a.loop.SetIcon(myTheme.RepeatIcon)
	}
}

func (a *AuxControls) OnShowPlayQueue(f func()) {
	a.showQueue.OnTapped = f
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

func (v *volumeSlider) Tapped(e *fyne.PointEvent) {
	v.Slider.Tapped(e)
	fyne.CurrentApp().Driver().CanvasForObject(v).Unfocus()
}

func (v *volumeSlider) MinSize() fyne.Size {
	return fyne.NewSize(v.Width, v.Slider.MinSize().Height)
}

func (v *volumeSlider) Scrolled(e *fyne.ScrollEvent) {
	v.SetValue(v.Value + float64(0.5*e.Scrolled.DY))
}

type VolumeControl struct {
	widget.BaseWidget

	icon   *IconButton
	slider *volumeSlider

	OnSetVolume func(int)

	muted   bool
	lastVol int

	container *fyne.Container
}

func NewVolumeControl(initialVol int) *VolumeControl {
	v := &VolumeControl{}
	v.ExtendBaseWidget(v)
	v.icon = NewIconButton(theme.VolumeUpIcon(), v.toggleMute)
	v.icon.SetToolTip("Mute")
	v.icon.IconSize = IconButtonSizeSmaller
	v.slider = NewVolumeSlider(100)
	v.lastVol = initialVol
	v.slider.Step = 1
	v.slider.Orientation = widget.Horizontal
	v.slider.Value = float64(v.lastVol)
	v.slider.OnChanged = v.onChanged
	v.container = container.NewHBox(container.NewCenter(v.icon), v.slider)
	return v
}

// Sets the volume that is displayed in the slider.
// Does not invoke OnSetVolume callback.
func (v *VolumeControl) SetVolume(vol int) {
	if (vol == v.lastVol && !v.muted) || (v.muted && vol == 0) {
		return
	}
	v.lastVol = vol
	v.muted = false
	v.setDisplayedVolume(vol)
}

func (v *VolumeControl) onChanged(volume float64) {
	vol := int(volume)
	v.lastVol = vol
	v.muted = false
	v.updateIconForVolume(vol)
	v.invokeOnVolumeChange(vol)
}

func (v *VolumeControl) toggleMute() {
	if !v.muted {
		v.muted = true
		v.lastVol = int(v.slider.Value)
		v.setDisplayedVolume(0)
		v.invokeOnVolumeChange(0)
	} else {
		v.muted = false
		v.setDisplayedVolume(v.lastVol)
		v.invokeOnVolumeChange(v.lastVol)
	}
}

func (v *VolumeControl) CreateRenderer() fyne.WidgetRenderer {
	v.ExtendBaseWidget(v)
	return widget.NewSimpleRenderer(v.container)
}

func (v *VolumeControl) setDisplayedVolume(vol int) {
	v.slider.Value = float64(vol)
	v.slider.Refresh()
	v.updateIconForVolume(vol)
}

func (v *VolumeControl) invokeOnVolumeChange(vol int) {
	if v.OnSetVolume != nil {
		v.OnSetVolume(vol)
	}
}

func (v *VolumeControl) updateIconForVolume(vol int) {
	if vol <= 0 {
		v.icon.SetIcon(theme.VolumeMuteIcon())
	} else if vol < 50 {
		v.icon.SetIcon(theme.VolumeDownIcon())
	} else {
		v.icon.SetIcon(theme.VolumeUpIcon())
	}
}
