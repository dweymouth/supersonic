package layouts

import "fyne.io/fyne/v2"

var _ fyne.Layout = (*CenterPadLayout)(nil)

type MaxPadLayout struct {
	PadLeft   float32
	PadRight  float32
	PadTop    float32
	PadBottom float32
}

// only supports one object
func (c *MaxPadLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(c.PadLeft+c.PadRight, c.PadTop+c.PadBottom)
	}
	return fyne.Size{
		Width:  objects[0].MinSize().Width + c.PadLeft + c.PadRight,
		Height: objects[0].MinSize().Height + c.PadTop + c.PadBottom,
	}
}

func (c *MaxPadLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	pos := fyne.NewPos(c.PadLeft, c.PadTop)
	objSize := fyne.NewSize(size.Width-c.PadLeft-c.PadRight, size.Height-c.PadTop-c.PadBottom)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		child.Move(pos)
		child.Resize(objSize)
	}
}
