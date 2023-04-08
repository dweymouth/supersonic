package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Similar to widgets.ToggleButtonGroup, but using text.
// The "active" label is bolded non-interactive text,
// while all the others are hyperlinks.
type ToggleText struct {
	widget.BaseWidget

	OnChanged func(int)

	labels         []string
	activeLabelIdx int
	container      *fyne.Container
}

func NewToggleText(activeLblIdx int, labels []string) *ToggleText {
	t := &ToggleText{labels: labels, activeLabelIdx: activeLblIdx}
	t.ExtendBaseWidget(t)
	t.container = container.NewHBox()
	for i, lbl := range labels {
		if i == activeLblIdx {
			t.container.Add(t.newBoldRichText(lbl))
		} else {
			hl := widget.NewHyperlink(lbl, nil)
			hl.OnTapped = t.buildOnTapped(i)
			t.container.Add(hl)
		}
	}
	return t
}

func (t *ToggleText) SetActivatedLabel(idx int) {
	changed := t.activeLabelIdx != idx
	// update old label to hyperlink
	hl := widget.NewHyperlink(t.labels[t.activeLabelIdx], nil)
	hl.OnTapped = t.buildOnTapped(t.activeLabelIdx)
	t.container.Objects[t.activeLabelIdx] = hl
	// update activated label to bold text
	t.container.Objects[idx] = t.newBoldRichText(t.labels[idx])
	t.activeLabelIdx = idx
	if changed {
		t.Refresh()
	}
}

func (t *ToggleText) buildOnTapped(i int) func() {
	return func() {
		t.onActivated(i)
	}
}

func (t *ToggleText) newBoldRichText(text string) *widget.RichText {
	return widget.NewRichText(&widget.TextSegment{
		Text:  text,
		Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}},
	})
}

func (t *ToggleText) onActivated(idx int) {
	changed := t.activeLabelIdx != idx
	t.SetActivatedLabel(idx)
	if changed && t.OnChanged != nil {
		t.OnChanged(t.activeLabelIdx)
	}
}

func (t *ToggleText) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}
