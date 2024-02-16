package widgets

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Tappable = (*ListRowBase)(nil)
var _ fyne.Widget = (*ListRowBase)(nil)
var _ fyne.Focusable = (*ListRowBase)(nil)

// Base type used for all list rows in widgets such as Tracklist, etc.
type ListRowBase struct {
	widget.BaseWidget

	Content  fyne.CanvasObject
	Selected bool
	Focused  bool

	OnTapped        func()
	OnDoubleTapped  func()
	OnFocusNeighbor func(up bool) //TODO: func(up, selecting bool)

	tappedAt      int64 // unixMillis
	focusedRect   *canvas.Rectangle
	selectionRect *canvas.Rectangle
}

// We implement our own double tapping so that the Tapped behavior
// can be triggered instantly.
func (l *ListRowBase) Tapped(*fyne.PointEvent) {
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

func (l *ListRowBase) FocusGained() {
	l.Focused = true
	l.Refresh()
}

func (l *ListRowBase) FocusLost() {
	l.Focused = false
	l.Refresh()
}

func (l *ListRowBase) TypedKey(e *fyne.KeyEvent) {
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

func (l *ListRowBase) TypedRune(r rune) {
}

func (l *ListRowBase) Refresh() {
	l.focusedRect.FillColor = theme.HoverColor()
	l.focusedRect.Hidden = !l.Focused
	l.selectionRect.FillColor = theme.SelectionColor()
	l.selectionRect.Hidden = !l.Selected
	l.BaseWidget.Refresh()
}

func (l *ListRowBase) CreateRenderer() fyne.WidgetRenderer {
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
