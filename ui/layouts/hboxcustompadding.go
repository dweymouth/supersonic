package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Layout = (*VboxCustomPadding)(nil)

type HboxCustomPadding struct {
	ExtraPad        float32
	DisableThemePad bool
}

func (*HboxCustomPadding) isSpacer(obj fyne.CanvasObject) bool {
	if !obj.Visible() {
		return false
	}
	if spacer, ok := obj.(layout.SpacerObject); ok {
		return spacer.ExpandHorizontal()
	}

	return false
}

func (h *HboxCustomPadding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := h.themePad() + h.ExtraPad
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		if h.isSpacer(child) {
			continue
		}

		minSize.Height = fyne.Max(child.MinSize().Height, minSize.Height)
		minSize.Width += child.MinSize().Width
		if addPadding {
			minSize.Width += padding
		}
		addPadding = true
	}
	return minSize
}

func (h *HboxCustomPadding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	total := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		if h.isSpacer(child) {
			spacers++
			continue
		}
		total += child.MinSize().Width
	}

	x, y := float32(0), float32(0)
	padding := h.themePad() + h.ExtraPad
	extra := size.Width - total - (padding * float32(len(objects)-spacers-1))
	extraCell := float32(0)
	if spacers > 0 {
		extraCell = extra / float32(spacers)
	}
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if h.isSpacer(child) {
			x += extraCell
		}
		width := child.MinSize().Width
		child.Move(fyne.NewPos(x, y))
		child.Resize(fyne.NewSize(width, size.Height))
		x += padding + width
	}
}

func (h *HboxCustomPadding) themePad() float32 {
	if h.DisableThemePad {
		return 0
	}
	return theme.Padding()
}
