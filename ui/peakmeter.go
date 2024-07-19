package ui

import (
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	meterRangeDB       = 60
	rmsSmoothingFactor = 0.8
	peakHoldFrames     = 60
)

type PeakFN func() (float64, float64, float64, float64)

type PeakMeter struct {
	widget.BaseWidget
	peakFnDB       PeakFN
	lPeak          float64
	rPeak          float64
	lRMS           float64
	rRMS           float64
	lPeakHold      float64
	rPeakHold      float64
	lPeakHoldFrame uint64
	rPeakHoldFrame uint64
	frameCounter   uint64

	lPeakRect     canvas.Rectangle
	rPeakRect     canvas.Rectangle
	lRMSRect      canvas.Rectangle
	rRMSRect      canvas.Rectangle
	lPeakHoldRect canvas.Rectangle
	rPeakHoldRect canvas.Rectangle
	anim          *fyne.Animation
}

func NewPeakMeter(peakFnDB PeakFN) *PeakMeter {
	p := &PeakMeter{peakFnDB: peakFnDB}
	p.ExtendBaseWidget(p)
	return p
}

func (p *PeakMeter) Start() {
	if p.anim != nil {
		return
	}
	p.anim = fyne.NewAnimation(time.Duration(math.MaxInt64) /*until stopped*/, p.tick)
	p.anim.Start()
}

func (p *PeakMeter) Stop() {
	if p.anim != nil {
		p.anim.Stop()
		p.anim = nil
		p.frameCounter = 0
		p.lPeakHoldFrame = 0
		p.rPeakHoldFrame = 0
	}
}

func (p *PeakMeter) tick(_ float32) {
	lPeak, rPeak, lRMS, rRMS := p.peakFnDB()
	p.lPeak = lPeak
	p.rPeak = rPeak
	lRMS = math.Max(-96, lRMS)
	rRMS = math.Max(-96, rRMS)
	p.lRMS = rmsSmoothingFactor*p.lRMS + (1-rmsSmoothingFactor)*lRMS
	p.rRMS = rmsSmoothingFactor*p.rRMS + (1-rmsSmoothingFactor)*rRMS

	if lPeak > p.lPeakHold || p.frameCounter-p.lPeakHoldFrame > peakHoldFrames {
		p.lPeakHold = lPeak
		p.lPeakHoldFrame = p.frameCounter
	}
	if rPeak > p.rPeakHold || p.frameCounter-p.rPeakHoldFrame > peakHoldFrames {
		p.rPeakHold = rPeak
		p.rPeakHoldFrame = p.frameCounter
	}

	p.frameCounter++
	p.Refresh()
}

func (p *PeakMeter) CreateRenderer() fyne.WidgetRenderer {
	c := theme.PrimaryColor().(color.NRGBA)
	c.A = 128
	p.lPeakRect.FillColor = c
	p.rPeakRect.FillColor = c
	p.lRMSRect.FillColor = c
	p.rRMSRect.FillColor = c
	p.lPeakHoldRect.FillColor = theme.ForegroundColor()
	p.rPeakHoldRect.FillColor = p.lPeakHoldRect.FillColor
	return widget.NewSimpleRenderer(
		container.New(
			&peakMeterLayout{p},
			&p.lPeakRect, &p.rPeakRect,
			&p.lRMSRect, &p.rRMSRect,
			&p.lPeakHoldRect, &p.rPeakHoldRect,
		),
	)
}

type peakMeterLayout struct {
	p *PeakMeter
}

func (l *peakMeterLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(50, 150)
}

func (l *peakMeterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	lPeakH := float32(math.Max(0, meterRangeDB+l.p.lPeak)/meterRangeDB) * size.Height
	rPeakH := float32(math.Max(0, meterRangeDB+l.p.rPeak)/meterRangeDB) * size.Height
	lRMSH := float32(math.Max(0, meterRangeDB+l.p.lRMS)/meterRangeDB) * size.Height
	rRMSH := float32(math.Max(0, meterRangeDB+l.p.rRMS)/meterRangeDB) * size.Height
	lPeakHoldPos := float32(math.Max(0, meterRangeDB+l.p.lPeakHold)/meterRangeDB) * size.Height
	rPeakHoldPos := float32(math.Max(0, meterRangeDB+l.p.rPeakHold)/meterRangeDB) * size.Height

	halfW := size.Width / 2
	objects[0].Move(fyne.NewPos(0, size.Height-lPeakH))
	objects[0].Resize(fyne.NewSize(halfW, lPeakH))
	objects[1].Move(fyne.NewPos(halfW, size.Height-rPeakH))
	objects[1].Resize(fyne.NewSize(halfW, rPeakH))
	objects[2].Move(fyne.NewPos(0, size.Height-lRMSH))
	objects[2].Resize(fyne.NewSize(halfW, lRMSH))
	objects[3].Move(fyne.NewPos(halfW, size.Height-rRMSH))
	objects[3].Resize(fyne.NewSize(halfW, rRMSH))

	peakHoldHt := theme.SeparatorThicknessSize() * 2
	objects[4].Move(fyne.NewPos(0, size.Height-lPeakHoldPos-peakHoldHt))
	objects[4].Resize(fyne.NewSize(halfW, peakHoldHt))
	objects[5].Move(fyne.NewPos(halfW, size.Height-rPeakHoldPos-peakHoldHt))
	objects[5].Resize(fyne.NewSize(halfW, peakHoldHt))
}
