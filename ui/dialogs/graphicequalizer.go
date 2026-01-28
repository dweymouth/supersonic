package dialogs

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

// EQ Presets for 15-band ISO equalizer (values in dB)
// Bands: 25, 40, 63, 100, 160, 250, 400, 630, 1k, 1.6k, 2.5k, 4k, 6.3k, 10k, 16k
type eqPreset struct {
	Name   string
	Preamp float64
	Bands  [15]float64
}

func getEqPresets() []eqPreset {
	return []eqPreset{
		{Name: lang.L("EQ Flat"), Preamp: 0, Bands: [15]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{Name: lang.L("EQ Rock"), Preamp: 0, Bands: [15]float64{5, 4, 3, 1, -1, -1, 0, 2, 3, 4, 4, 4, 3, 2, 2}},
		{Name: lang.L("EQ Pop"), Preamp: 0, Bands: [15]float64{-1, -1, 0, 2, 4, 4, 2, 0, -1, -1, 0, 1, 2, 3, 3}},
		{Name: lang.L("EQ Jazz"), Preamp: 0, Bands: [15]float64{4, 3, 1, 2, -2, -2, 0, 2, 3, 3, 3, 4, 4, 4, 4}},
		{Name: lang.L("EQ Classical"), Preamp: 0, Bands: [15]float64{5, 4, 3, 2, -1, -1, 0, 2, 3, 3, 3, 2, 2, 2, -1}},
		{Name: lang.L("EQ Bass Boost"), Preamp: 0, Bands: [15]float64{6, 5, 4, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{Name: lang.L("EQ Treble Boost"), Preamp: 0, Bands: [15]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 4, 5, 6, 6}},
		{Name: lang.L("EQ Vocal"), Preamp: 0, Bands: [15]float64{-2, -3, -3, 1, 4, 4, 4, 3, 2, 1, 0, -1, -2, -2, -3}},
		{Name: lang.L("EQ Electronic"), Preamp: 0, Bands: [15]float64{5, 4, 2, 0, -2, -2, 0, 2, 3, 4, 4, 3, 4, 4, 3}},
		{Name: lang.L("EQ Acoustic"), Preamp: 0, Bands: [15]float64{5, 4, 3, 1, 2, 1, 1, 2, 2, 2, 1, 2, 2, 3, 2}},
		{Name: lang.L("EQ R&B"), Preamp: 0, Bands: [15]float64{3, 6, 5, 2, -2, -2, 2, 3, 2, 2, 3, 3, 3, 3, 4}},
		{Name: lang.L("EQ Loudness"), Preamp: 0, Bands: [15]float64{6, 5, 3, 0, -1, -1, -1, -1, 0, 1, 2, 4, 5, 5, 3}},
	}
}

type GraphicEqualizer struct {
	widget.BaseWidget

	OnChanged       func(band int, gain float64)
	OnPreampChanged func(gain float64)

	bandSliders  []*eqSlider
	preampSlider *eqSlider
	presetSelect *widget.Select
	container    *fyne.Container
	eqPresets    []eqPreset
}

func NewGraphicEqualizer(preamp float64, bandFreqs []string, bandGains []float64) *GraphicEqualizer {
	g := &GraphicEqualizer{}
	g.ExtendBaseWidget(g)
	g.buildSliders(preamp, bandFreqs, bandGains)

	return g
}

func (g *GraphicEqualizer) buildSliders(preamp float64, bands []string, bandGains []float64) {
	// Build preset selector
	g.eqPresets = getEqPresets()
	presetNames := make([]string, len(g.eqPresets))
	for i, p := range g.eqPresets {
		presetNames[i] = p.Name
	}
	g.presetSelect = widget.NewSelect(presetNames, func(selected string) {
		for _, p := range g.eqPresets {
			if p.Name == selected {
				g.applyPreset(p)
				break
			}
		}
	})
	g.presetSelect.PlaceHolder = lang.L("EQ Preset")

	// Reset button
	resetBtn := widget.NewButton(lang.L("Reset"), func() {
		g.applyPreset(g.eqPresets[0]) // Apply "Flat" preset
		g.presetSelect.SetSelected(lang.L("EQ Flat"))
	})

	// Top bar with preset and reset
	topBar := container.NewHBox(
		widget.NewLabel(lang.L("EQ Preset:")),
		g.presetSelect,
		layout.NewSpacer(),
		resetBtn,
	)

	// Range labels
	rng := container.NewVBox(
		newCaptionTextSizeLabel("+12", fyne.TextAlignTrailing),
		layout.NewSpacer(),
		newCaptionTextSizeLabel("0 dB", fyne.TextAlignTrailing),
		layout.NewSpacer(),
		newCaptionTextSizeLabel("-12", fyne.TextAlignTrailing),
	)

	g.bandSliders = make([]*eqSlider, len(bands))
	bandSlidersCtr := container.New(layouts.NewGridLayoutWithColumnsAndPadding(len(bands)+2, -16))

	// Preamp slider
	pre := newCaptionTextSizeLabel(lang.L("EQ Preamp"), fyne.TextAlignCenter)
	g.preampSlider = newEQSlider()
	g.preampSlider.SetValue(preamp)
	g.preampSlider.OnChanged = func(f float64) {
		if g.OnPreampChanged != nil {
			g.OnPreampChanged(f)
		}
		g.preampSlider.UpdateToolTip()
	}
	g.preampSlider.UpdateToolTip()
	bandSlidersCtr.Add(container.NewBorder(nil, pre, nil, nil, g.preampSlider))
	bandSlidersCtr.Add(container.NewBorder(nil, widget.NewLabel(""), nil, nil, rng))

	// Band sliders
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

	sliderArea := container.NewStack(
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

	g.container = container.NewBorder(topBar, nil, nil, nil, sliderArea)
}

func (g *GraphicEqualizer) applyPreset(preset eqPreset) {
	// Apply preamp
	g.preampSlider.SetValue(preset.Preamp)
	g.preampSlider.UpdateToolTip()
	if g.OnPreampChanged != nil {
		g.OnPreampChanged(preset.Preamp)
	}

	// Apply band gains
	for i, gain := range preset.Bands {
		if i < len(g.bandSliders) {
			g.bandSliders[i].SetValue(gain)
			g.bandSliders[i].UpdateToolTip()
			if g.OnChanged != nil {
				g.OnChanged(i, gain)
			}
		}
	}
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
