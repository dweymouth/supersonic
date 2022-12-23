package ui

import (
	"fmt"
	"supersonic/backend"
	"supersonic/player"

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
	curTimeLabel   *widget.Label
	totalTimeLabel *widget.Label
	prev           *widget.Button
	playpause      *widget.Button
	next           *widget.Button
	container      *fyne.Container

	totalTime       float64
	player          *player.Player
	playbackManager *backend.PlaybackManager
}

var _ fyne.Widget = (*PlayerControls)(nil)

// NewPlayerControls sets up the seek bar, and transport buttons, and returns the encompassing Container.
func NewPlayerControls(p *player.Player) *PlayerControls {
	pc := &PlayerControls{player: p}
	pc.ExtendBaseWidget(pc)

	pc.slider = NewTrackPosSlider()
	pc.curTimeLabel = widget.NewLabel("0:00")
	pc.totalTimeLabel = widget.NewLabel("0:00")

	pc.slider.OnDragEnd = func(f float64) {
		p.Seek(fmt.Sprintf("%d", int(f*100)), player.SeekAbsolutePercent)
	}
	pc.slider.OnChanged = func(f float64) {
		time := f * pc.totalTime
		pc.curTimeLabel.SetText(SecondsToTimeString(time))
	}

	pc.prev = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		p.SeekBackOrPrevious()
	})
	pc.next = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		p.SeekNext()
	})
	pc.playpause = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		p.PlayPause()
	})

	p.OnPaused(func() {
		pc.playpause.SetIcon(theme.MediaPlayIcon())
	})
	p.OnPlaying(func() {
		pc.playpause.SetIcon(theme.MediaPauseIcon())
	})
	p.OnStopped(func() {
		pc.playpause.SetIcon(theme.MediaPlayIcon())
	})

	buttons := container.NewHBox(pc.prev, pc.playpause, pc.next)
	b := container.New(layout.NewCenterLayout(), buttons)

	c := container.NewBorder(nil, nil, pc.curTimeLabel, pc.totalTimeLabel, pc.slider)
	pc.container = container.NewVBox(c, b)

	return pc
}

func (pc *PlayerControls) SetPlaybackManager(pm *backend.PlaybackManager) {
	pc.playbackManager = pm
	pm.OnPlayTimeUpdate(func(curTime float64, totalTime float64) {
		pc.doPlayTimeUpdate(curTime, totalTime)
	})
}

func (pc *PlayerControls) doPlayTimeUpdate(curTime, totalTime float64) {
	// TODO: there is a bug with very long tracks (~20min) where the
	// curtime label will bounce back and forth +- 1sec (rounding issue?)
	pc.totalTime = totalTime
	if !pc.playbackManager.IsSeeking() {
		v := 0.0
		if totalTime > 0 {
			v = curTime / totalTime
		}

		updated := false
		tt := SecondsToTimeString(totalTime)
		if tt != pc.totalTimeLabel.Text {
			pc.totalTimeLabel.SetText(tt)
			updated = true
		}
		if !pc.slider.IsDragging() {
			ct := SecondsToTimeString(curTime)
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
}

func (p *PlayerControls) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}
