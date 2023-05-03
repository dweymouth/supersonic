package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var _ fyne.Layout = (*VboxCustomPadding)(nil)

type HboxCustomPadding struct {
	ExtraPad        float32
	DisableThemePad bool
}

func (v *HboxCustomPadding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		minSize.Height = fyne.Max(child.MinSize().Height, minSize.Height)
		minSize.Width += child.MinSize().Width
	}
	minSize.Width += (v.themePad() + v.ExtraPad) * float32(len(objects)-1)
	return minSize
}

func (v *HboxCustomPadding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	total := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		total += child.MinSize().Width
	}

	x, y := float32(0), float32(0)

	padding := v.themePad() + v.ExtraPad
	extra := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		width := child.MinSize().Width
		child.Move(fyne.NewPos(x+extra, y))
		x += width
		child.Resize(fyne.NewSize(width, size.Height))
		extra += padding
	}
}

func (v *HboxCustomPadding) themePad() float32 {
	if v.DisableThemePad {
		return 0
	}
	return theme.Padding()
}
