package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Layout = (*VboxCustomPadding)(nil)

type VboxCustomPadding struct {
	ExtraPad float32
}

func (*VboxCustomPadding) isSpacer(obj fyne.CanvasObject) bool {
	spacer, ok := obj.(layout.SpacerObject)
	return ok && spacer.ExpandVertical()
}

func (v *VboxCustomPadding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := theme.Padding() + v.ExtraPad
	for _, child := range objects {
		if !child.Visible() || v.isSpacer(child) {
			continue
		}

		childMin := child.MinSize()
		minSize.Width = fyne.Max(childMin.Width, minSize.Width)
		minSize.Height += childMin.Height
		if addPadding {
			minSize.Height += padding
		}
		addPadding = true
	}
	return minSize
}

func (v *VboxCustomPadding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	visibleObjects := 0
	// Size taken up by visible objects
	total := float32(0)

	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if v.isSpacer(child) {
			spacers++
			continue
		}

		visibleObjects++
		total += child.MinSize().Height
	}

	padding := theme.Padding() + v.ExtraPad

	// Amount of space not taken up by visible objects and inter-object padding
	extra := size.Height - total - (padding * float32(visibleObjects-1))

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

		if v.isSpacer(child) {
			y += spacerSize
			continue
		}
		child.Move(fyne.NewPos(x, y))

		height := child.MinSize().Height
		y += padding + height
		child.Resize(fyne.NewSize(size.Width, height))
	}
}
