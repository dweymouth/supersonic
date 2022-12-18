package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Layout = (*VboxCustomPadding)(nil)

type VboxCustomPadding struct {
	ExtraPad float32
}

func (v *VboxCustomPadding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		minSize.Width = fyne.Max(child.MinSize().Width, minSize.Width)
		minSize.Height += child.MinSize().Height
	}
	minSize.Height += (theme.Padding() + v.ExtraPad) * float32(len(objects)-1)
	return minSize
}

func (v *VboxCustomPadding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	total := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		total += child.MinSize().Height
	}

	x, y := float32(0), float32(0)

	extra := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		height := child.MinSize().Height
		child.Move(fyne.NewPos(x, y+extra))
		y += height
		child.Resize(fyne.NewSize(size.Width, height))
		extra += (theme.Padding() + v.ExtraPad)
	}
}
