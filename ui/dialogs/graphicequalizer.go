package dialogs

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type GraphicEqualizer struct {
	widget.BaseWidget

	OnChanged       func(band int, gain float64)
	OnPreampChanged func(gain float64)

	bandSliders []*eqSlider
	container   *fyne.Container
}

func NewGraphicEqualizer(preamp float64, bandFreqs []string, bandGains []float64) *GraphicEqualizer {
	g := &GraphicEqualizer{}
	g.ExtendBaseWidget(g)
	g.buildSliders(preamp, bandFreqs, bandGains)

	return g
}

func (g *GraphicEqualizer) buildSliders(preamp float64, bands []string, bandGains []float64) {
	rng := container.NewVBox(
		newCaptionTextSizeLabel("+12", fyne.TextAlignTrailing),
		layout.NewSpacer(),
		newCaptionTextSizeLabel("0", fyne.TextAlignTrailing),
		layout.NewSpacer(),
		newCaptionTextSizeLabel("-12", fyne.TextAlignTrailing),
	)
	g.bandSliders = make([]*eqSlider, len(bands))
	bandSlidersCtr := container.New(layouts.NewGridLayoutWithColumnsAndPadding(len(bands)+2, -16))
	pre := newCaptionTextSizeLabel("Pre", fyne.TextAlignCenter)
	preampSlider := newEQSlider()
	preampSlider.SetValue(preamp)
	preampSlider.OnChanged = func(f float64) {
		if g.OnPreampChanged != nil {
			g.OnPreampChanged(f)
		}
		preampSlider.UpdateToolTip()
	}
	preampSlider.UpdateToolTip()
	bandSlidersCtr.Add(container.NewBorder(nil, pre, nil, nil, preampSlider))
	bandSlidersCtr.Add(container.NewBorder(nil, widget.NewLabel(""), nil, nil, rng))
	for i, band := range bands {
		s := newEQSlider()
		if i < len(bandGains) {
			s.SetValue(bandGains[i])
			s.UpdateToolTip()
		}
		_i := i
		s.OnChanged = func(f float64) {
			if g.OnChanged != nil {
				g.OnChanged(_i, f)
			}
			g.bandSliders[_i].UpdateToolTip()
		}
		l := newCaptionTextSizeLabel(band, fyne.TextAlignCenter)
		c := container.NewBorder(nil, l, nil, nil, s)
		bandSlidersCtr.Add(c)
		g.bandSliders[i] = s
	}
	g.container = container.NewStack(
		container.NewBorder(nil, widget.NewLabel(""), nil, nil,
			container.NewBorder(nil, nil, util.NewHSpace(5), util.NewHSpace(5),
				container.NewVBox(
					layout.NewSpacer(),
					myTheme.NewThemedRectangle(theme.ColorNameInputBackground),
					layout.NewSpacer(),
				),
			),
		),
		bandSlidersCtr,
	)
}

func newCaptionTextSizeLabel(text string, alignment fyne.TextAlign) *widget.RichText {
	l := widget.NewRichTextWithText(text)
	ts := l.Segments[0].(*widget.TextSegment)
	ts.Style.SizeName = theme.SizeNameCaptionText
	ts.Style.Alignment = alignment
	return l
}

func (g *GraphicEqualizer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

type eqSlider struct {
	ttwidget.Slider
}

func newEQSlider() *eqSlider {
	s := &eqSlider{
		Slider: ttwidget.Slider{
			Slider: widget.Slider{
				Orientation: widget.Vertical,
				Min:         -12,
				Max:         12,
				Step:        0.1,
			},
		},
	}
	s.UpdateToolTip()
	s.ExtendBaseWidget(s)
	return s
}

func (s *eqSlider) UpdateToolTip() {
	s.SetToolTip(fmt.Sprintf("%0.1f dB", s.Value))
}

// We implement our own double tapping so that the Tapped behavior
// can be triggered instantly.
func (s *eqSlider) DoubleTapped(e *fyne.PointEvent) {
	s.SetValue(0)
}
