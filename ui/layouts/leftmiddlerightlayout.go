package layouts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// Lays out up to 3 objects such that the middle object, Objects[1],
// is centered in the available space and takes up a fixed width.
// The left, and right (if non-nil), split the leftover space equally.
type LeftMiddleRightLayout struct {
	middleWidth float32
	hbox        fyne.Layout
}

func NewLeftMiddleRightLayout(middleWidth float32) *LeftMiddleRightLayout {
	return &LeftMiddleRightLayout{
		middleWidth: middleWidth,
		hbox:        layout.NewHBoxLayout(),
	}
}

func (b *LeftMiddleRightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	hboxSize := b.hbox.MinSize(objects)
	return fyne.Size{
		Height: hboxSize.Height,
		Width:  hboxSize.Width + fyne.Max(0, b.middleWidth-objects[1].MinSize().Width),
	}
}

func (b *LeftMiddleRightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	midW := fyne.Max(b.middleWidth, objects[1].MinSize().Width)
	lrW := (size.Width - midW - theme.Padding()*4) / 2
	objects[0].Resize(fyne.NewSize(lrW, size.Height))
	objects[0].Move(fyne.NewPos(theme.Padding(), 0))
	objects[1].Resize(fyne.NewSize(midW, size.Height))
	objects[1].Move(fyne.NewPos(lrW+theme.Padding()*2, 0))
	if objects[2] != nil {
		objects[2].Resize(fyne.NewSize(lrW, size.Height))
		objects[2].Move(fyne.NewPos(lrW+midW+theme.Padding()*3, 0))
	}
}
