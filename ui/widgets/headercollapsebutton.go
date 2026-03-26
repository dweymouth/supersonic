package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HeaderCollapseButton struct {
	widget.Button

	Collapsed bool

	shouldHide bool
}

func NewHeaderCollapseButton(onTapped func()) *HeaderCollapseButton {
	b := &HeaderCollapseButton{
		Button: widget.Button{
			Icon:       theme.ContentRemoveIcon(),
			Importance: widget.LowImportance,
		},
	}
	b.OnTapped = func() {
		b.Collapsed = !b.Collapsed
		onTapped()
	}
	b.ExtendBaseWidget(b)
	return b
}

// HideIfNotMousedIn hides the button after a short delay
// if the mouse is not hovering over it.
func (b *HeaderCollapseButton) HideIfNotMousedIn() {
	b.shouldHide = true
	fyne.Do(func() {
		if b.shouldHide {
			b.Hide()
			b.shouldHide = false
		}
	})
}

func (b *HeaderCollapseButton) MouseIn(e *desktop.MouseEvent) {
	b.shouldHide = false
	b.Button.MouseIn(e)
}

func (b *HeaderCollapseButton) MinSize() fyne.Size {
	return fyne.NewSize(24, 24)
}

func (b *HeaderCollapseButton) Refresh() {
	if b.Collapsed {
		b.Icon = theme.ContentAddIcon()
	} else {
		b.Icon = theme.ContentRemoveIcon()
	}
	b.Button.Refresh()
}
