package widgets

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

// The "aux" controls for playback, positioned to the right
// of the BottomPanel. Currently only volume control.
type AuxControls struct {
	widget.BaseWidget

	OnChangeAutoplay func(autoplay bool)

	VolumeControl *VolumeControl
	quality       *ttwidget.Button
	autoplay      *IconButton
	cast          *IconButton
	showQueue     *IconButton

	qualityInfo QualityPathInfo
	qualityPop  *widget.PopUp
	container   *fyne.Container
}

type QualityPathInfo struct {
	Badge                string
	Status               string
	SourceFormat         string
	DecodePath           string
	OutputPath           string
	DACFormat            string
	DeviceName           string
	DeviceTransport      string
	PlaybackPath         string
	Reason               string
	ExclusiveActive      bool
	BitPerfectActive     bool
	OutputMixable        bool
	PhysicalFormatCount  int
	ExclusiveFormatCount int
	DeviceMinSampleRate  int
	DeviceMaxSampleRate  int
	DeviceMaxBitDepth    int
	DeviceChannels       int
	SourceIsDSD          bool
	DSDRate              int
	DoPCarrierRate       int
}

func NewAuxControls(initialVolume int, initialAutoplay bool) *AuxControls {
	a := &AuxControls{
		VolumeControl: NewVolumeControl(initialVolume),
		quality:       ttwidget.NewButton("Audio", nil),
		autoplay:      NewIconButton(myTheme.AutoplayIcon, nil),
		cast:          NewIconButton(myTheme.CastIcon, nil),
		showQueue:     NewIconButton(myTheme.PlayQueueIcon, nil),
	}
	a.quality.Importance = widget.LowImportance
	a.quality.OnTapped = a.showQualityPath

	a.cast.IconSize = IconButtonSizeSmaller
	a.cast.SetToolTip(lang.L("Cast to device"))

	a.autoplay.Highlighted = initialAutoplay
	// a.autoplay.IconSize = IconButtonSizeSmaller
	a.autoplay.SetToolTip(lang.L("Autoplay"))
	a.autoplay.OnTapped = func() {
		a.SetAutoplay(!a.autoplay.Highlighted)
		if a.OnChangeAutoplay != nil {
			a.OnChangeAutoplay(a.autoplay.Highlighted)
		}
	}

	a.showQueue.IconSize = IconButtonSizeSmaller
	a.showQueue.SetToolTip(lang.L("Show play queue"))

	a.container = container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			layout.NewSpacer(),
			container.NewHBox(layout.NewSpacer(), a.quality, layout.NewSpacer()),
			a.VolumeControl,
			container.New(
				layout.NewCustomPaddedHBoxLayout(theme.Padding()*1.5),
				layout.NewSpacer(), a.autoplay, a.cast, a.showQueue, util.NewHSpace(5)),
			layout.NewSpacer(),
		),
	)
	return a
}

func (a *AuxControls) SetQualityPath(info QualityPathInfo) {
	if info.Badge == "" {
		info.Badge = "Audio"
	}
	if info.Status == "" {
		info.Status = "Shared Output"
	}
	a.qualityInfo = info
	a.quality.SetText(info.Badge)
	a.quality.SetToolTip(a.qualityToolTip(info))
	if a.qualityPop != nil {
		fynetooltip.DestroyPopUpToolTipLayer(a.qualityPop)
		a.qualityPop.Hide()
		a.qualityPop = nil
	}
	a.Refresh()
}

func (a *AuxControls) qualityToolTip(info QualityPathInfo) string {
	parts := []string{info.Status}
	if info.DeviceName != "" {
		parts = append(parts, info.DeviceName)
	}
	if info.Reason != "" && !info.BitPerfectActive {
		parts = append(parts, info.Reason)
	}
	return strings.Join(parts, " - ")
}

func (a *AuxControls) showQualityPath() {
	canvas := fyne.CurrentApp().Driver().CanvasForObject(a.quality)
	if canvas == nil {
		return
	}
	info := a.qualityInfo
	title := widget.NewLabelWithStyle(info.Status, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	rows := []fyne.CanvasObject{title, widget.NewSeparator()}
	addRow := func(label, value string) {
		if value == "" {
			return
		}
		rows = append(rows, container.NewGridWithColumns(2,
			widget.NewLabel(label),
			widget.NewLabel(value),
		))
	}
	addRow("Source", info.SourceFormat)
	addRow("Decode", info.DecodePath)
	addRow("Output", info.OutputPath)
	addRow("DAC format", info.DACFormat)
	addRow("Device", strings.TrimSpace(strings.Join([]string{info.DeviceName, info.DeviceTransport}, " ")))
	addRow("Backend", info.PlaybackPath)
	if info.DeviceMaxSampleRate > 0 || info.DeviceMaxBitDepth > 0 || info.PhysicalFormatCount > 0 {
		addRow("DAC capability", fmt.Sprintf("%d-bit / %.1f kHz, %d physical (%d exclusive)",
			info.DeviceMaxBitDepth,
			float64(info.DeviceMaxSampleRate)/1000,
			info.PhysicalFormatCount,
			info.ExclusiveFormatCount))
	}
	if info.SourceIsDSD && info.DoPCarrierRate > 0 {
		addRow("DoP carrier", fmt.Sprintf("%.1f kHz", float64(info.DoPCarrierRate)/1000))
	}
	if info.Reason != "" && !info.BitPerfectActive {
		addRow("Reason", info.Reason)
	}
	content := container.NewPadded(container.NewVBox(rows...))
	a.qualityPop = widget.NewPopUp(content, canvas)
	fynetooltip.AddPopUpToolTipLayer(a.qualityPop)
	pos := fyne.NewPos(0, -content.MinSize().Height-theme.Padding())
	a.qualityPop.ShowAtRelativePosition(pos, a.quality)
}

func (a *AuxControls) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

func (a *AuxControls) DisableCastButton() {
	a.cast.Disable()
}

func (a *AuxControls) SetIsRemotePlayer(isRemote bool) {
	a.cast.Enable()
	a.cast.Highlighted = isRemote
	a.cast.Refresh()
}

func (a *AuxControls) SetAutoplay(autoplay bool) {
	if autoplay == a.autoplay.Highlighted {
		return
	}
	a.autoplay.Highlighted = autoplay
	a.autoplay.Refresh()
}

func (a *AuxControls) OnShowPlayQueue(f func()) {
	a.showQueue.OnTapped = f
}

func (a *AuxControls) OnShowCastMenu(f func(func())) {
	a.cast.OnTapped = func() {
		f(a.DisableCastButton)
	}
}

type volumeSlider struct {
	ttwidget.Slider

	Width float32
}

func NewVolumeSlider(width float32) *volumeSlider {
	v := &volumeSlider{
		Slider: ttwidget.Slider{
			Slider: widget.Slider{
				Min:         0,
				Max:         100,
				Step:        1,
				Orientation: widget.Horizontal,
				Value:       100,
			},
		},
		Width: width,
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *volumeSlider) Tapped(e *fyne.PointEvent) {
	v.Slider.Tapped(e)
	if c := fyne.CurrentApp().Driver().CanvasForObject(v); c != nil {
		c.Unfocus()
	}
}

func (v *volumeSlider) MinSize() fyne.Size {
	return fyne.NewSize(v.Width, v.Slider.MinSize().Height)
}

func (v *volumeSlider) Scrolled(e *fyne.ScrollEvent) {
	v.SetValue(v.Value + math.Max(-10, math.Min(10, float64(0.5*e.Scrolled.DY))))
}

type VolumeControl struct {
	widget.BaseWidget

	icon   *IconButton
	slider *volumeSlider

	OnSetVolume func(int)

	muted   bool
	lastVol int

	setVolDebouncer func()
	delaySetVolume  bool
	pendingVolume   int

	softwareVolumeLocked bool
	volumeLockReason     string

	container *fyne.Container
}

func NewVolumeControl(initialVol int) *VolumeControl {
	v := &VolumeControl{}
	v.ExtendBaseWidget(v)
	v.icon = NewIconButton(theme.VolumeUpIcon(), v.toggleMute)
	v.icon.SetToolTip(lang.L("Mute"))
	v.icon.IconSize = IconButtonSizeSmaller
	v.slider = NewVolumeSlider(100)
	v.lastVol = initialVol
	v.slider.Step = 1
	v.slider.Orientation = widget.Horizontal
	v.slider.Value = float64(v.lastVol)
	v.slider.OnChanged = v.onChanged

	// for players that are slow to respond to volume changes
	// (e.g. DLNA), delay responding to the SetVolume call
	// to avoid hiccuping from callback "echoes"
	v.setVolDebouncer = util.NewDebouncer(100*time.Millisecond, func() {
		v.delaySetVolume = false
		v.SetVolume(v.pendingVolume)
	})

	v.container = container.NewHBox(container.NewCenter(v.icon), v.slider)
	return v
}

func (v *VolumeControl) SoftwareVolumeLocked() bool {
	return v.softwareVolumeLocked
}

func (v *VolumeControl) SetSoftwareVolumeLocked(locked bool, reason string) {
	if v.softwareVolumeLocked == locked && v.volumeLockReason == reason {
		return
	}
	v.softwareVolumeLocked = locked
	v.volumeLockReason = reason

	if locked {
		v.delaySetVolume = false
		v.setDisplayedVolume(100)
		v.icon.Disable()
		v.slider.Disable()
		v.icon.SetToolTip(reason)
		v.slider.SetToolTip(reason)
	} else {
		if v.muted {
			v.setDisplayedVolume(0)
		} else {
			v.setDisplayedVolume(v.lastVol)
		}
		v.icon.Enable()
		v.slider.Enable()
		v.icon.SetToolTip(lang.L("Mute"))
		v.slider.SetToolTip("")
	}
	v.Refresh()
}

// Sets the volume that is displayed in the slider.
// Does not invoke OnSetVolume callback.
func (v *VolumeControl) SetVolume(vol int) {
	if v.delaySetVolume {
		v.pendingVolume = vol
		return
	}

	if v.softwareVolumeLocked {
		v.lastVol = vol
		v.muted = false
		v.setDisplayedVolume(100)
		return
	}

	if (vol == v.lastVol && !v.muted) || (v.muted && vol == 0) {
		return
	}
	v.lastVol = vol
	v.muted = false
	v.setDisplayedVolume(vol)
}

func (v *VolumeControl) onChanged(volume float64) {
	if v.softwareVolumeLocked {
		v.setDisplayedVolume(100)
		return
	}
	vol := int(volume)
	v.delaySetVolume = true
	v.setVolDebouncer()
	v.lastVol = vol
	v.muted = false
	v.updateIconForVolume(vol)
	v.invokeOnVolumeChange(vol)
}

func (v *VolumeControl) toggleMute() {
	if v.softwareVolumeLocked {
		return
	}
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
