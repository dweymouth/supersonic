package controller

import (
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"github.com/dweymouth/supersonic/ui/visualizations"
)

// embedded in parent controller struct
type visualizationData struct {
	peakMeter *visualizations.PeakMeter

	visualizationAnim *fyne.Animation
}

func (c *Controller) InitVisualizations() {
	c.App.LocalPlayer.OnStopped(c.stopVisualizationAnim)
	c.App.LocalPlayer.OnPaused(c.stopVisualizationAnim)
	c.App.LocalPlayer.OnPlaying(func() {
		if c.peakMeter != nil {
			c.startVisualizationAnim()
		}
	})
}

func (c *Controller) ShowPeakMeter() {
	if c.peakMeter != nil {
		return
	}
	win := fyne.CurrentApp().NewWindow(lang.L("Peak Meter"))
	win.SetCloseIntercept(func() {
		c.stopVisualizationAnim()
		c.peakMeter = nil
		win.Close()
	})
	c.peakMeter = visualizations.NewPeakMeter()
	win.SetContent(c.peakMeter)
	c.startVisualizationAnim()
	win.Show()
}

func (c *Controller) stopVisualizationAnim() {
	if c.visualizationAnim != nil {
		c.visualizationAnim.Stop()
		c.visualizationAnim = nil
		c.App.LocalPlayer.SetPeaksEnabled(false)
	}
}

func (c *Controller) startVisualizationAnim() {
	if c.visualizationAnim == nil {
		c.App.LocalPlayer.SetPeaksEnabled(true)
		c.visualizationAnim = fyne.NewAnimation(
			time.Duration(math.MaxInt64), /*until stopped*/
			c.tickVisualizations)
		c.visualizationAnim.Start()
	}
}

func (c *Controller) tickVisualizations(_ float32) {
	lP, rP, lRMS, rRMS := c.App.LocalPlayer.GetPeaks()
	if c.visualizationData.peakMeter != nil {
		c.visualizationData.peakMeter.UpdatePeaks(lP, rP, lRMS, rRMS)
	}
}
