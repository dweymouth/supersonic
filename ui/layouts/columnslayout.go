package layouts

import "fyne.io/fyne/v2"

// ColumnsLayout lays out a number of items into columns.
// There are two types of columns: fixed-width and variable width.
// A fixed width column is any with a non-negative width and will
// be laid out with that width. A variable width column is created
// by using any negative number as its width. Variable width columns
// are all laid out with the same width, splitting the "leftover" space
// equally between themselves after accounting for the fixed-width columns.
// Hidden items are not shown and take up 0 space.
type ColumnsLayout struct {
	ColumnWidths []float32
}

func NewColumnsLayout(widths []float32) *ColumnsLayout {
	return &ColumnsLayout{ColumnWidths: widths}
}

func (c *ColumnsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var width float32
	var height float32
	for i := 0; i < len(objects); i++ {
		if !objects[i].Visible() {
			continue
		}
		height = fyne.Max(height, objects[i].MinSize().Height)
		w := objects[i].MinSize().Width
		if i < len(c.ColumnWidths) && c.ColumnWidths[i] > w {
			w = c.ColumnWidths[i]
		}
		width += w
	}
	return fyne.NewSize(width, height)
}

func (c *ColumnsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	var fixedW float32
	var expandObjCount int
	for i := 0; i < min(len(objects), len(c.ColumnWidths)); i++ {
		if !objects[i].Visible() {
			continue
		}
		if c.ColumnWidths[i] < 0 {
			expandObjCount++
		} else {
			itemW := objects[i].MinSize().Width
			fixedW += fyne.Max(itemW, c.ColumnWidths[i])
		}
	}
	extraW := size.Width - fixedW
	expandObjW := extraW / float32(expandObjCount)

	var x float32
	for i := 0; i < len(objects); i++ {
		if !objects[i].Visible() {
			continue
		}
		w := objects[i].MinSize().Width
		if i < len(c.ColumnWidths) && c.ColumnWidths[i] > w {
			w = c.ColumnWidths[i]
		} else if c.ColumnWidths[i] < 0 && expandObjW > w {
			w = expandObjW
		}
		objects[i].Resize(fyne.NewSize(w, size.Height))
		objects[i].Move(fyne.NewPos(x, 0))
		x += w
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
