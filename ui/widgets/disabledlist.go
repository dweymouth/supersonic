package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// DisabledList extends List to be disabled so that the
// focus manager considers it unfocusable.
// This is needed since we handle row focusability ourselves.
type DisabledList struct {
	widget.List
}

func NewDisabledList(len func() int, create func() fyne.CanvasObject, update func(widget.GridWrapItemID, fyne.CanvasObject)) *DisabledList {
	g := &DisabledList{
		List: widget.List{
			Length:     len,
			CreateItem: create,
			UpdateItem: update,
		},
	}
	g.ExtendBaseWidget(g)
	return g
}

var _ fyne.Disableable = (*DisabledList)(nil)

func (g *DisabledList) Disabled() bool { return true }

func (g *DisabledList) Disable() {}

func (g *DisabledList) Enable() {}
