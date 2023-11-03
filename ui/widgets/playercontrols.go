package widgets

import (
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TrackPosSlider is a custom slider that exposes an additional
// IsDragging() API as well as some other customizations
type TrackPosSlider struct {
	widget.Slider

	// to avoid "data echoes" when slider value is updated as
	// playback position changes
	IgnoreNextChangeEnded bool

	isDragging bool
}

func NewTrackPosSlider() *TrackPosSlider {
	slider := &TrackPosSlider{
		Slider: widget.Slider{
			Value:       0,
			Min:         0,
			Max:         1,
			Step:        0.002,
			Orientation: widget.Horizontal,
		},
	}
	slider.ExtendBaseWidget(slider)
	return slider
}

func (t *TrackPosSlider) SetValue(value float64) {
	t.IgnoreNextChangeEnded = true
	t.Slider.SetValue(value)
}

func (t *TrackPosSlider) Tapped(e *fyne.PointEvent) {
	t.isDragging = false
	t.IgnoreNextChangeEnded = false
	t.Slider.Tapped(e)

	// don't keep focus after being tapped
	fyne.CurrentApp().Driver().CanvasForObject(t).Focus(nil)
}

// override to increase the distance moved by keyboard control
func (t *TrackPosSlider) TypedKey(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyLeft:
		t.Slider.SetValue(t.Value - 0.05)
	case fyne.KeyRight:
		t.Slider.SetValue(t.Value + 0.05)
	default:
		t.Slider.TypedKey(e)
	}
}

func (t *TrackPosSlider) DragEnd() {
	t.isDragging = false
	t.IgnoreNextChangeEnded = false
	t.Slider.DragEnd()
}

func (t *TrackPosSlider) Dragged(e *fyne.DragEvent) {
	t.isDragging = true
	t.Slider.Dragged(e)
}

func (t *TrackPosSlider) IsDragging() bool {
	return t.isDragging
}

type PlayerControls struct {
	widget.BaseWidget

	slider         *TrackPosSlider
	curTimeLabel   *labelMinSize
	totalTimeLabel *labelMinSize
	prev           *widget.Button
	playpause      *widget.Button
	next           *widget.Button
	container      *fyne.Container

	totalTime float64
}

var _ fyne.Widget = (*PlayerControls)(nil)

type labelMinSize struct {
	widget.Label
	MinWidth float32
}

func (l *labelMinSize) MinSize() fyne.Size {
	return fyne.NewSize(l.MinWidth, l.Label.MinSize().Height)
}

func NewLabelMinSize(text string, minWidth float32) *labelMinSize {
	l := &labelMinSize{MinWidth: minWidth, Label: widget.Label{Text: text}}
	l.ExtendBaseWidget(l)
	return l
}

// NewPlayerControls sets up the seek bar, and transport buttons.
func NewPlayerControls() *PlayerControls {
	pc := &PlayerControls{}
	pc.ExtendBaseWidget(pc)

	pc.slider = NewTrackPosSlider()
	pc.curTimeLabel = NewLabelMinSize(util.SecondsToTimeString(0), 55)
	pc.curTimeLabel.Alignment = fyne.TextAlignTrailing
	pc.totalTimeLabel = NewLabelMinSize(util.SecondsToTimeString(0), 55)
	pc.totalTimeLabel.Alignment = fyne.TextAlignTrailing

	pc.slider.OnChanged = func(f float64) {
		if pc.slider.IsDragging() {
			time := f * pc.totalTime
			pc.curTimeLabel.SetText(util.SecondsToTimeString(time))
		}
	}

	pc.prev = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {})
	pc.next = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {})
	pc.playpause = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {})

	buttons := container.NewHBox(pc.prev, pc.playpause, pc.next)
	b := container.New(layout.NewCenterLayout(), buttons)

	c := container.NewBorder(nil, nil, pc.curTimeLabel, pc.totalTimeLabel, pc.slider)
	pc.container = container.NewVBox(c, b)

	return pc
}

func (pc *PlayerControls) OnSeek(f func(float64)) {
	pc.slider.OnChangeEnded = func(pos float64) {
		if pc.slider.IgnoreNextChangeEnded {
			pc.slider.IgnoreNextChangeEnded = false
		} else {
			f(pos)
		}
	}
}

func (pc *PlayerControls) OnSeekPrevious(f func()) {
	pc.prev.OnTapped = f
}

func (pc *PlayerControls) OnSeekNext(f func()) {
	pc.next.OnTapped = f
}

func (pc *PlayerControls) OnPlayPause(f func()) {
	pc.playpause.OnTapped = f
}

func (pc *PlayerControls) SetPlaying(playing bool) {
	if playing {
		pc.playpause.SetIcon(theme.MediaPauseIcon())
	} else {
		pc.playpause.SetIcon(theme.MediaPlayIcon())
	}
}

func (pc *PlayerControls) UpdatePlayTime(curTime, totalTime float64) {
	pc.totalTime = totalTime
	v := 0.0
	if totalTime > 0 {
		v = curTime / totalTime
	}

	updated := false
	tt := util.SecondsToTimeString(totalTime)
	if tt != pc.totalTimeLabel.Text {
		pc.totalTimeLabel.SetText(tt)
		updated = true
	}
	if !pc.slider.IsDragging() {
		ct := util.SecondsToTimeString(curTime)
		if ct != pc.curTimeLabel.Text {
			pc.curTimeLabel.SetText(ct)
			updated = true
		}
		if updated {
			// Only update slider once a second when time label changes
			pc.slider.SetValue(v)
		}
	}
}

func (p *PlayerControls) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}
