package widgets

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	list "github.com/dweymouth/fyne-advanced-list"
)

// FocusList extends List to be disabled so that the focus manager
// considers it unfocusable, and adds utilities for handling our
// own focus navigation on the rows directly (with FocusListRow
type FocusList struct {
	list.List

	mutex sync.Mutex
}

type FocusListRow interface {
	fyne.Focusable
	ItemID() widget.ListItemID
	SetItemID(widget.ListItemID)

	SetOnTapped(func())
	SetOnDoubleTapped(func())
	SetOnFocusNeighbor(func(up bool))
}

func NewFocusList(len func() int, create func() fyne.CanvasObject, update func(widget.GridWrapItemID, fyne.CanvasObject)) *FocusList {
	g := &FocusList{
		List: list.List{
			HideSeparators: true,
			Length:         len,
			CreateItem:     create,
			UpdateItem:     update,
		},
	}
	g.ExtendBaseWidget(g)
	return g
}

var _ fyne.Disableable = (*FocusList)(nil)

func (g *FocusList) Disabled() bool { return true }

func (g *FocusList) Disable() {}

func (g *FocusList) Enable() {}

func (g *FocusList) FocusNeighbor(curItem widget.ListItemID, up bool) {
	focusIdx := curItem + 1
	if up {
		focusIdx = curItem - 1
	}
	if focusIdx >= 0 && focusIdx < g.Length() {
		g.ScrollTo(focusIdx)
	}
	g.mutex.Lock()
	other := g.ItemForID(focusIdx)
	g.mutex.Unlock()
	if other != nil {
		fyne.CurrentApp().Driver().CanvasForObject(g).Focus(other.(fyne.Focusable))
	}
}

var _ fyne.Tappable = (*FocusListRowBase)(nil)
var _ fyne.Widget = (*FocusListRowBase)(nil)
var _ fyne.Focusable = (*FocusListRowBase)(nil)

// Base type used for all list rows in widgets such as Tracklist, etc.
type FocusListRowBase struct {
	widget.BaseWidget

	ListItemID widget.ListItemID
	Content    fyne.CanvasObject
	Selected   bool
	Focused    bool

	OnTapped        func()
	OnDoubleTapped  func()
	OnFocusNeighbor func(up bool) //TODO: func(up, selecting bool)

	tappedAt      int64 // unixMillis
	focusedRect   *canvas.Rectangle
	selectionRect *canvas.Rectangle
}

func (l *FocusListRowBase) SetOnTapped(f func()) {
	l.OnTapped = f
}

func (l *FocusListRowBase) SetOnDoubleTapped(f func()) {
	l.OnDoubleTapped = f
}

func (l *FocusListRowBase) SetOnFocusNeighbor(f func(up bool)) {
	l.OnFocusNeighbor = f
}

func (l *FocusListRowBase) ItemID() widget.ListItemID {
	return l.ListItemID
}

func (l *FocusListRowBase) SetItemID(id widget.ListItemID) {
	l.ListItemID = id
}

func (l *FocusListRowBase) EnsureUnfocused() {
	if l.Focused {
		c := fyne.CurrentApp().Driver().CanvasForObject(l)
		if c != nil {
			c.Unfocus()
		}
	}
	l.Focused = false
}

// We implement our own double tapping so that the Tapped behavior
// can be triggered instantly.
func (l *FocusListRowBase) Tapped(*fyne.PointEvent) {
	prevTap := l.tappedAt
	l.tappedAt = time.Now().UnixMilli()
	if l.tappedAt-prevTap < 300 {
		if l.OnDoubleTapped != nil {
			l.OnDoubleTapped()
		}
	} else {
		if l.OnTapped != nil {
			l.OnTapped()
		}
	}
}

func (l *FocusListRowBase) FocusGained() {
	l.Focused = true
	l.Refresh()
}

func (l *FocusListRowBase) FocusLost() {
	l.Focused = false
	l.Refresh()
}

func (l *FocusListRowBase) TypedKey(e *fyne.KeyEvent) {
	/**
	// TODO: enable shift+arrows for selection, but it's complicated to implement in the widgets
	desktop, ok := fyne.CurrentApp().Driver().(desktop.Driver)
	isSelecting := func() bool {
		return ok && desktop.CurrentKeyModifiers()&fyne.KeyModifierShift != 0
	}
	*/
	switch {
	case e.Name == fyne.KeyUp:
		if l.OnFocusNeighbor != nil {
			l.OnFocusNeighbor(true)
		}
	case e.Name == fyne.KeyDown:
		if l.OnFocusNeighbor != nil {
			l.OnFocusNeighbor(false)
		}
	case e.Name == fyne.KeySpace:
		if l.OnTapped != nil {
			l.OnTapped()
		}
	case e.Name == fyne.KeyReturn || e.Name == fyne.KeyEnter:
		if l.OnDoubleTapped != nil {
			l.OnDoubleTapped()
		} else if l.OnTapped != nil {
			l.OnTapped()
		}
	}
}

func (l *FocusListRowBase) TypedRune(r rune) {
}

func (l *FocusListRowBase) Refresh() {
	l.focusedRect.FillColor = theme.HoverColor()
	l.focusedRect.Hidden = !l.Focused
	l.selectionRect.FillColor = theme.SelectionColor()
	l.selectionRect.Hidden = !l.Selected
	l.BaseWidget.Refresh()
}

func (l *FocusListRowBase) CreateRenderer() fyne.WidgetRenderer {
	if l.selectionRect == nil {
		l.selectionRect = canvas.NewRectangle(theme.SelectionColor())
		l.selectionRect.CornerRadius = theme.SelectionRadiusSize()
		l.selectionRect.Hidden = !l.Selected
	}
	if l.focusedRect == nil {
		l.focusedRect = canvas.NewRectangle(theme.HoverColor())
		l.focusedRect.CornerRadius = theme.SelectionRadiusSize()
		l.focusedRect.Hidden = !l.Focused
	}
	return widget.NewSimpleRenderer(
		container.NewStack(l.selectionRect, l.focusedRect, l.Content),
	)
}
