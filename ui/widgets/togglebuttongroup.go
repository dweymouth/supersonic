package widgets

import (
	"supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Lays out multiple buttons horizontally with no padding,
// and uses button.Importance to highlight exactly one which is "active".
// Similar to a segmented control in other UI toolkits.
type ToggleButtonGroup struct {
	widget.BaseWidget

	buttonContainer *fyne.Container

	activeBtnIdx int
}

func NewToggleButtonGroup(activatedBtnIdx int, buttons ...*widget.Button) *ToggleButtonGroup {
	t := &ToggleButtonGroup{}
	t.ExtendBaseWidget(t)
	t.buttonContainer = container.New(&layouts.HboxCustomPadding{DisableThemePad: true})
	for i, b := range buttons {
		b.Importance = widget.MediumImportance
		t.buttonContainer.Add(b)
		prevOnTapped := b.OnTapped
		b.OnTapped = func(i int) func() {
			return func() {
				if t.onTapped(i) && prevOnTapped != nil {
					prevOnTapped()
				}
			}
		}(i)
	}
	if activatedBtnIdx >= 0 && activatedBtnIdx <= len(buttons) {
		buttons[activatedBtnIdx].Importance = widget.HighImportance
	}

	return t
}

func (t *ToggleButtonGroup) ActivatedButtonIndex() int {
	return t.activeBtnIdx
}

func (t *ToggleButtonGroup) SetActivatedButton(idx int) {
	changed := t.activeBtnIdx != idx
	t.activeBtnIdx = idx
	for i, b := range t.buttonContainer.Objects {
		if i == idx {
			b.(*widget.Button).Importance = widget.HighImportance
		} else {
			b.(*widget.Button).Importance = widget.MediumImportance
		}
	}
	if changed {
		t.Refresh()
	}
}

func (t *ToggleButtonGroup) onTapped(btnIdx int) bool {
	changed := t.activeBtnIdx != btnIdx
	t.SetActivatedButton(btnIdx)
	return changed
}

func (t *ToggleButtonGroup) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.buttonContainer)
}
