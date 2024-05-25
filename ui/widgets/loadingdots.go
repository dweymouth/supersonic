package widgets

import (
	"context"
	"image/color"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LoadingDots struct {
	widget.BaseWidget

	running    atomic.Bool
	animCancel context.CancelFunc

	dots      [3]minSizeCircle
	animPos   int
	container *fyne.Container
}

func NewLoadingDots() *LoadingDots {
	l := &LoadingDots{}
	for i := range l.dots {
		l.dots[i] = minSizeCircle{Circle: canvas.Circle{
			FillColor: theme.DisabledColor()},
		}
	}
	l.ExtendBaseWidget(l)
	l.Hide()
	return l
}

func (l *LoadingDots) Start() {
	if !l.running.CompareAndSwap(false, true) {
		return // already started
	}
	for i := range l.dots {
		l.dots[i].Circle.FillColor = theme.DisabledColor()
	}
	l.animPos = 0
	l.Show()
	var ctx context.Context
	ctx, l.animCancel = context.WithCancel(context.Background())
	go l.animate(ctx)
}

func (l *LoadingDots) Stop() {
	if !l.running.Load() {
		return
	}
	l.Hide()
	l.animCancel()
}

func (l *LoadingDots) animate(ctx context.Context) {
	foreground := theme.ForegroundColor()
	disabled := theme.DisabledColor()
	l.doTick(foreground, disabled)
	ticker := time.NewTicker(333 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			l.running.Store(false)
			return
		case <-ticker.C:
			l.doTick(foreground, disabled)
		}
	}
}

func (l *LoadingDots) doTick(foreground, disabled color.Color) {
	oldDot := l.animPos - 1
	if oldDot == -1 {
		oldDot = len(l.dots) - 1
	}
	l.dots[l.animPos].Circle.FillColor = foreground
	l.dots[oldDot].Circle.FillColor = disabled
	l.Refresh()
	l.animPos += 1
	if l.animPos >= len(l.dots) {
		l.animPos = 0
	}
}

func (l *LoadingDots) CreateRenderer() fyne.WidgetRenderer {
	if l.container == nil {
		layout := layout.NewCustomPaddedLayout(0, 0, 3, 3)
		l.container = container.NewHBox(
			container.New(layout, &l.dots[0]),
			container.New(layout, &l.dots[1]),
			container.New(layout, &l.dots[2]),
		)
	}
	return widget.NewSimpleRenderer(l.container)
}

type minSizeCircle struct {
	widget.BaseWidget
	Circle canvas.Circle
}

func (m *minSizeCircle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(&m.Circle)
}

func (m *minSizeCircle) MinSize() fyne.Size {
	return fyne.NewSquareSize(theme.IconInlineSize() / 2)
}
