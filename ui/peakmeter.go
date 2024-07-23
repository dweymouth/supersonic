package ui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

const (
	meterRangeDB       = 62
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

	anim *fyne.Animation
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
	return newPeakMeterRenderer(p)
}

type peakMeterRenderer struct {
	p *PeakMeter

	lLabel        canvas.Text
	rLabel        canvas.Text
	lPeakRect     canvas.Rectangle
	rPeakRect     canvas.Rectangle
	lRMSRect      canvas.Rectangle
	rRMSRect      canvas.Rectangle
	lPeakHoldRect canvas.Rectangle
	rPeakHoldRect canvas.Rectangle

	rulerLines  []canvas.Rectangle
	rulerLabels []canvas.Text

	fgColor   color.Color
	bgColor   color.Color
	ruleColor color.Color
}

func newPeakMeterRenderer(pm *PeakMeter) *peakMeterRenderer {
	p := &peakMeterRenderer{p: pm}
	p.lLabel.Text = "L"
	p.rLabel.Text = "R"
	numRules := int(math.Ceil(float64(meterRangeDB) / 10))
	p.rulerLines = make([]canvas.Rectangle, numRules)
	p.rulerLabels = make([]canvas.Text, numRules)
	x := 0
	for i := range p.rulerLabels {
		p.rulerLabels[i].Text = fmt.Sprintf("%d dB", x)
		p.rulerLabels[i].Resize(p.rulerLabels[i].MinSize())
		x -= 10
	}

	return p
}

func (l *peakMeterRenderer) MinSize() fyne.Size {
	return fyne.NewSize(275, 75)
}

func (l *peakMeterRenderer) Layout(size fyne.Size) {
	topSpacing := float32(5)
	lrLabelWidth := float32(20)
	overflowWidth := float32(10)
	ruleLabelHeight := float32(10)
	meterWidth := size.Width - lrLabelWidth - overflowWidth - topSpacing

	lPeakWidth := float32(math.Max(0, meterRangeDB+l.p.lPeak)/meterRangeDB) * meterWidth
	rPeakWidth := float32(math.Max(0, meterRangeDB+l.p.rPeak)/meterRangeDB) * meterWidth
	lRMSWidth := float32(math.Max(0, meterRangeDB+l.p.lRMS)/meterRangeDB) * meterWidth
	rRMSWidth := float32(math.Max(0, meterRangeDB+l.p.rRMS)/meterRangeDB) * meterWidth
	lPeakHoldPos := float32(math.Max(0, meterRangeDB+l.p.lPeakHold)/meterRangeDB) * meterWidth
	rPeakHoldPos := float32(math.Max(0, meterRangeDB+l.p.rPeakHold)/meterRangeDB) * meterWidth

	barSpacing := float32(2)
	barHeight := size.Height/2 - barSpacing - ruleLabelHeight

	labelMin := l.lLabel.MinSize()
	l.lLabel.Move(fyne.NewPos(4, (barHeight-labelMin.Height)/2+topSpacing))
	l.lLabel.Resize(l.lLabel.MinSize())
	l.rLabel.Move(fyne.NewPos(4, barHeight+barSpacing+topSpacing+(barHeight-labelMin.Height)/2))

	l.lPeakRect.Move(fyne.NewPos(lrLabelWidth, topSpacing))
	l.lPeakRect.Resize(fyne.NewSize(lPeakWidth, barHeight))
	l.rPeakRect.Move(fyne.NewPos(lrLabelWidth, barHeight+barSpacing+topSpacing))
	l.rPeakRect.Resize(fyne.NewSize(rPeakWidth, barHeight))
	l.lRMSRect.Move(fyne.NewPos(lrLabelWidth, topSpacing))
	l.lRMSRect.Resize(fyne.NewSize(lRMSWidth, barHeight))
	l.rRMSRect.Move(fyne.NewPos(lrLabelWidth, barHeight+barSpacing+topSpacing))
	l.rRMSRect.Resize(fyne.NewSize(rRMSWidth, barHeight))

	peakHoldWidth := theme.SeparatorThicknessSize() * 2
	l.lPeakHoldRect.Move(fyne.NewPos(lPeakHoldPos+lrLabelWidth, topSpacing))
	l.lPeakHoldRect.Resize(fyne.NewSize(peakHoldWidth, barHeight))
	l.rPeakHoldRect.Move(fyne.NewPos(rPeakHoldPos+lrLabelWidth, barHeight+barSpacing+topSpacing))
	l.rPeakHoldRect.Resize(fyne.NewSize(peakHoldWidth, barHeight))

	ruleWidth := peakHoldWidth * 0.667
	x := lrLabelWidth + meterWidth
	for i := range l.rulerLines {
		bottom := (barHeight + barSpacing) * 2
		l.rulerLines[i].Move(fyne.NewPos(x, topSpacing))
		l.rulerLines[i].Resize(fyne.NewSize(ruleWidth, bottom))
		l.rulerLabels[i].Move(fyne.NewPos(x-10, bottom+topSpacing))
		x -= meterWidth * (10 / float32(meterRangeDB))
	}
}

func (l *peakMeterRenderer) Refresh() {
	foreground := theme.ForegroundColor()
	background := theme.BackgroundColor()
	errC := theme.ErrorColor()
	c := theme.PrimaryColor().(color.NRGBA)
	c.A = 128
	l.lLabel.Color = foreground
	l.rLabel.Color = foreground
	l.lLabel.TextSize = 16
	l.rLabel.TextSize = 16
	l.lLabel.TextStyle.Bold = true
	l.rLabel.TextStyle.Bold = true
	l.lPeakRect.FillColor = c
	l.rPeakRect.FillColor = c
	l.lRMSRect.FillColor = c
	l.rRMSRect.FillColor = c

	if l.p.lPeakHold >= -0.00001 {
		l.lPeakHoldRect.FillColor = errC
	} else {
		l.lPeakHoldRect.FillColor = foreground
	}
	if l.p.rPeakHold >= -0.00001 {
		l.rPeakHoldRect.FillColor = errC
	} else {
		l.rPeakHoldRect.FillColor = foreground
	}

	if foreground != l.fgColor || background != l.bgColor {
		l.ruleColor = myTheme.BlendColors(foreground, background, 0.5)
		l.fgColor = foreground
		l.bgColor = background
	}
	for i := range l.rulerLines {
		l.rulerLines[i].FillColor = l.ruleColor
		l.rulerLabels[i].TextSize = 11
	}

	l.Layout(l.p.Size())
}

func (l *peakMeterRenderer) Objects() []fyne.CanvasObject {
	obj := make([]fyne.CanvasObject, 0, 6+len(l.rulerLines))
	for i := range l.rulerLines {
		obj = append(obj, &l.rulerLines[i], &l.rulerLabels[i])
	}
	return append(obj,
		&l.lLabel, &l.rLabel,
		&l.lPeakRect, &l.rPeakRect,
		&l.lRMSRect, &l.rRMSRect,
		&l.lPeakHoldRect, &l.rPeakHoldRect)
}

func (l *peakMeterRenderer) Destroy() {
	l.p.Stop()
}
