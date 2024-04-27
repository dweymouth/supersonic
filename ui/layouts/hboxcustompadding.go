package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Layout = (*HboxCustomPadding)(nil)

type HboxCustomPadding struct {
	ExtraPad        float32
	DisableThemePad bool
}

func (*HboxCustomPadding) isSpacer(obj fyne.CanvasObject) bool {
	spacer, ok := obj.(layout.SpacerObject)
	return ok && spacer.ExpandHorizontal()
}

func (h *HboxCustomPadding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := h.themePad() + h.ExtraPad
	for _, child := range objects {
		if !child.Visible() || h.isSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Height = fyne.Max(childMin.Height, minSize.Height)
		minSize.Width += childMin.Width
		if addPadding {
			minSize.Width += padding
		}
		addPadding = true
	}
	return minSize
}

func (h *HboxCustomPadding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by visible objects
	total := float32(0)

	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if h.isSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		total += child.MinSize().Width
	}

	padding := h.themePad() + h.ExtraPad

	// Amount of space not taken up by visible objects and inter-object padding
	extra := size.Width - total - (padding * float32(visibleObjects-1))

	// Spacers split extra space equally
	spacerSize := float32(0)
	if spacers > 0 {
		spacerSize = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if h.isSpacer(child) {
			x += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		width := child.MinSize().Width
		x += padding + width
		child.Resize(fyne.NewSize(width, size.Height))
	}
}

func (h *HboxCustomPadding) themePad() float32 {
	if h.DisableThemePad {
		return 0
	}
	return theme.Padding()
}
