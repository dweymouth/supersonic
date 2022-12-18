package ui

import (
	"fmt"
	"gomuse/backend"
	"gomuse/player"

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

// NewPlayerControls sets up the seek bar, and transport buttons, and returns the encompassing Container.
func NewPlayerControls(p *player.Player, pm *backend.PlaybackManager) *fyne.Container {
	slider := NewTrackPosSlider()
	curTimeLabel := widget.NewLabel("0:00")
	totalTimeLabel := widget.NewLabel("0:00")

	c := container.NewBorder(nil, nil, curTimeLabel, totalTimeLabel, slider)

	slider.OnDragEnd = func(f float64) {
		p.Seek(fmt.Sprintf("%d", int(f*100)), player.SeekAbsolutePercent)
	}

	prev := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		p.SeekBackOrPrevious()
	})
	next := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		p.SeekNext()
	})
	playpause := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		p.PlayPause()
	})

	p.OnPaused(func() {
		playpause.SetIcon(theme.MediaPlayIcon())
	})
	p.OnPlaying(func() {
		playpause.SetIcon(theme.MediaPauseIcon())
	})
	p.OnStopped(func() {
		playpause.SetIcon(theme.MediaPlayIcon())
	})

	buttons := container.NewHBox(prev, playpause, next)
	b := container.New(layout.NewCenterLayout(), buttons)
	content := container.NewVBox(c, b)

	pm.OnPlayTimeUpdate(func(curTime float64, totalTime float64) {
		if !pm.IsSeeking() {
			v := 0.0
			if totalTime > 0 {
				v = curTime / totalTime
			}
			ct := SecondsToTimeString(curTime)
			updated := false
			if ct != curTimeLabel.Text {
				curTimeLabel.SetText(ct)
				updated = true
			}
			tt := SecondsToTimeString(totalTime)
			if tt != totalTimeLabel.Text {
				totalTimeLabel.SetText(tt)
				updated = true
			}
			if !slider.IsDragging() {
				if totalTime < 210 || updated {
					// if current track is long, we only need to redraw the slider
					// when the time label updates, to reduce screen redraws.
					slider.SetValue(v)
				}
			}
		}
	})

	return content
}
