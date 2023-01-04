package layouts

import (
	"fyne.io/fyne/v2"
)

var _ fyne.Layout = (*CenterPadLayout)(nil)

type CenterPadLayout struct {
	PadLeftRight float32
	PadTopBottom float32
}

// only supports one object
func (c *CenterPadLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return fyne.Size{
		Width:  objects[0].MinSize().Width + c.PadLeftRight*2,
		Height: objects[0].MinSize().Height + c.PadTopBottom*2,
	}
}

func (c *CenterPadLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objSize := objects[0].MinSize()
	xOffs := (size.Width - objSize.Width) / 2
	yOffs := (size.Height - objSize.Height) / 2
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		child.Move(fyne.NewPos(xOffs, yOffs))
		child.Resize(child.MinSize())
	}
}
