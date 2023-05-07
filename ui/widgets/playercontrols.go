package widgets

import (
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TrackPosSlider is a custom slider that doesn't trigger
// the seek action until drag end.
type TrackPosSlider struct {
	widget.Slider

	OnDragEnd func(float64)

	isDragging       bool
	lastDraggedValue float64
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

func (t *TrackPosSlider) DragEnd() {
	t.isDragging = false
	t.Slider.DragEnd()
	if t.OnDragEnd != nil {
		t.OnDragEnd(t.lastDraggedValue)
	}
}

func (t *TrackPosSlider) Dragged(e *fyne.DragEvent) {
	t.isDragging = true
	t.Slider.Dragged(e)
	t.lastDraggedValue = t.Value
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
	pc.slider.OnDragEnd = f
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

// This code will be OBSOLETE in Fyne 2.4
// which will natively add Tappable behavior to slider
// Tapped is called when a pointer tapped event is captured.
//
// Since: 2.4
func (t *TrackPosSlider) Tapped(e *fyne.PointEvent) {
	ratio := t.getRatio(e)
	val := t.Min + ratio*(t.Max-t.Min)
	t.isDragging = true
	t.SetValue(val)
	t.lastDraggedValue = val
	t.isDragging = false
	t.DragEnd()
}

func (t *TrackPosSlider) endOffset() float32 {
	return (theme.IconInlineSize()-4)/2 + theme.InnerPadding() - 1.5 // align with radio icons
}

func (t *TrackPosSlider) getRatio(e *fyne.PointEvent) float64 {
	pad := t.endOffset()

	x := e.Position.X
	y := e.Position.Y

	switch t.Orientation {
	case widget.Vertical:
		if y > t.Size().Height-pad {
			return 0.0
		} else if y < pad {
			return 1.0
		} else {
			return 1 - float64(y-pad)/float64(t.Size().Height-pad*2)
		}
	case widget.Horizontal:
		if x > t.Size().Width-pad {
			return 1.0
		} else if x < pad {
			return 0.0
		} else {
			return float64(x-pad) / float64(t.Size().Width-pad*2)
		}
	}
	return 0.0
}
