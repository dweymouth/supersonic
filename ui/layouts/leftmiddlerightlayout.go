package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// Lays out up to 3 objects such that the middle object, Objects[1],
// is centered in the available space and takes up a fixed width,
// or optionally a fraction of the overall space.
// The left, and right split the leftover space equally.
type LeftMiddleRightLayout struct {
	middleWidthMin      float32
	middleWidthFraction float32
	hbox                fyne.Layout
}

func NewLeftMiddleRightLayout(middleWidthMin, middleWidthFraction float32) *LeftMiddleRightLayout {
	return &LeftMiddleRightLayout{
		middleWidthMin:      middleWidthMin,
		middleWidthFraction: middleWidthFraction,
		hbox:                layout.NewHBoxLayout(),
	}
}

func (b *LeftMiddleRightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	hboxSize := b.hbox.MinSize(objects)
	return fyne.Size{
		Height: hboxSize.Height,
		Width:  hboxSize.Width + fyne.Max(0, b.middleWidthMin-objects[1].MinSize().Width),
	}
}

func (b *LeftMiddleRightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pad := theme.Padding()
	lrMinWidth := fyne.Max(objects[0].MinSize().Width, objects[2].MinSize().Width)
	midMinWidth := fyne.Max(b.middleWidthMin, objects[1].MinSize().Width)
	midMaxWidth := size.Width - lrMinWidth*2 - pad*4
	midW := fyne.Min(midMaxWidth, fyne.Max(midMinWidth, b.middleWidthFraction*size.Width))
	lrW := (size.Width - midW - pad*4) / 2
	objects[0].Resize(fyne.NewSize(lrW, size.Height))
	objects[0].Move(fyne.NewPos(pad, 0))
	objects[1].Resize(fyne.NewSize(midW, size.Height))
	objects[1].Move(fyne.NewPos(lrW+pad*2, 0))
	objects[2].Resize(fyne.NewSize(lrW, size.Height))
	objects[2].Move(fyne.NewPos(lrW+midW+pad*3, 0))
}
