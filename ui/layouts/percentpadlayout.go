package layouts

import "fyne.io/fyne/v2"

// Centers and pads a given item by a percent (0-1)
// of the width and height that should be taken by the objects
type PercentPadLayout struct {
	LeftRightObjectPercent float32
	TopBottomObjectPercent float32
}

func (l *PercentPadLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var objMinSize fyne.Size
	for _, obj := range objects {
		objMinSize = objMinSize.Max(obj.MinSize())
	}
	if objMinSize.IsZero() {
		return objMinSize
	}

	return fyne.Size{
		Width:  objMinSize.Width / l.LeftRightObjectPercent,
		Height: objMinSize.Height / l.LeftRightObjectPercent,
	}
}

func (l *PercentPadLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	objSize := fyne.NewSize(
		size.Width*l.LeftRightObjectPercent,
		size.Height*l.TopBottomObjectPercent,
	)
	objPos := fyne.NewPos(
		size.Width*(1-l.LeftRightObjectPercent)/2,
		size.Height*(1-l.TopBottomObjectPercent)/2,
	)
	for _, obj := range objects {
		obj.Resize(objSize)
		obj.Move(objPos)
	}
}
