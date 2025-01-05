package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ToastOverlay struct {
	widget.BaseWidget

	currentToast     *toast
	currentToastAnim *fyne.Animation

	container *fyne.Container
}

func NewToastOverlay() *ToastOverlay {
	t := &ToastOverlay{container: container.NewWithoutLayout()}
	t.container.Objects = make([]fyne.CanvasObject, 0, 1)
	t.ExtendBaseWidget(t)
	return t
}

func (t *ToastOverlay) ShowSuccessToast(message string) {
	t.cancelPreviousToast()

	t.currentToast = newToast(false, message)
	t.container.Objects = append(t.container.Objects, t.currentToast)

	s := t.Size()
	min := t.currentToast.MinSize()
	pad := theme.Padding()
	t.currentToast.Resize(min)
	endPos := fyne.NewPos(s.Width-min.Width-pad, s.Height-min.Height-pad)
	startPos := fyne.NewPos(s.Width, endPos.Y)
	t.currentToastAnim = canvas.NewPositionAnimation(startPos, endPos, 100*time.Millisecond, func(p fyne.Position) {
		if ct := t.currentToast; ct != nil {
			ct.Move(p)
		}
		if p == endPos {
			t.currentToastAnim = nil
		}
	})
	t.currentToastAnim.Curve = fyne.AnimationEaseOut
	t.currentToastAnim.Start()
	t.Refresh()
}

func (t *ToastOverlay) Resize(size fyne.Size) {
	if t.currentToast != nil && t.currentToastAnim == nil {
		// move to the anchor position
		min := t.currentToast.MinSize()
		pad := theme.Padding()
		t.currentToast.Move(fyne.NewPos(size.Width-min.Width-pad, size.Height-min.Height-pad))
	} // else if animation is running -- well, hope this doesn't happen ;)

	t.BaseWidget.Resize(size)
}

func (t *ToastOverlay) cancelPreviousToast() {
	if t.currentToast == nil {
		return
	}
	if t.currentToastAnim != nil {
		t.currentToastAnim.Stop()
		t.currentToastAnim = nil
	}
	t.container.Objects = t.container.Objects[:0]
	t.currentToast = nil
}

func (t *ToastOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}

type toast struct {
	widget.BaseWidget

	isErr   bool
	message string
}

func newToast(isErr bool, message string) *toast {
	t := &toast{isErr: isErr, message: message}
	t.ExtendBaseWidget(t)
	return t
}

func (t *toast) CreateRenderer() fyne.WidgetRenderer {
	return newToastRenderer(t)
}

// swallow all tap/mouse events because toast is transparent
var (
	_ fyne.Tappable          = (*toast)(nil)
	_ fyne.SecondaryTappable = (*toast)(nil)
	_ desktop.Hoverable      = (*toast)(nil)
	_ desktop.Mouseable      = (*toast)(nil)
)

func (*toast) Tapped(*fyne.PointEvent)          {}
func (*toast) TappedSecondary(*fyne.PointEvent) {}
func (*toast) MouseIn(*desktop.MouseEvent)      {}
func (*toast) MouseOut()                        {}
func (*toast) MouseMoved(*desktop.MouseEvent)   {}
func (*toast) MouseUp(*desktop.MouseEvent)      {}
func (*toast) MouseDown(*desktop.MouseEvent)    {}

type toastRenderer struct {
	container   *fyne.Container
	background  *canvas.Rectangle
	accent      *canvas.Rectangle
	accentColor fyne.ThemeColorName
}

func newToastRenderer(t *toast) *toastRenderer {
	title := lang.L("Success")
	accentColor := theme.ColorNamePrimary
	if t.isErr {
		title = lang.L("Error")
		accentColor = theme.ColorNameError
	}

	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	background := canvas.NewRectangle(th.Color(theme.ColorNameOverlayBackground, v))
	background.CornerRadius = th.Size(theme.SizeNameInputRadius)
	background.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	background.StrokeWidth = th.Size(theme.SizeNameInputBorder) * 2
	accent := canvas.NewRectangle(th.Color(accentColor, v))
	accent.SetMinSize(fyne.NewSize(4, 1))

	pad := theme.Padding()
	return &toastRenderer{
		background:  background,
		accent:      accent,
		accentColor: accentColor,
		container: container.NewStack(
			background,
			container.New(&layout.CustomPaddedLayout{
				TopPadding:    2 * pad,
				BottomPadding: 2 * pad,
				LeftPadding:   2 * pad,
				RightPadding:  pad,
			},
				container.NewBorder(nil, nil, accent, nil,
					widget.NewRichText(
						&widget.TextSegment{Text: title, Style: widget.RichTextStyleSubHeading},
						&widget.TextSegment{Text: t.message},
					)),
			),
		),
	}
}

var _ fyne.WidgetRenderer = (*toastRenderer)(nil)

func (*toastRenderer) Destroy() {}

func (t *toastRenderer) Layout(s fyne.Size) {
	t.container.Layout.Layout(t.container.Objects, s)
}

func (t *toastRenderer) MinSize() fyne.Size {
	return t.container.MinSize()
}

func (t *toastRenderer) Objects() []fyne.CanvasObject {
	return t.container.Objects
}

func (t *toastRenderer) Refresh() {
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	t.background.FillColor = th.Color(theme.ColorNameOverlayBackground, v)
	t.background.CornerRadius = th.Size(theme.SizeNameInputRadius)
	t.background.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	t.background.StrokeWidth = th.Size(theme.SizeNameInputBorder) * 2
	t.background.Refresh()

	t.accent.FillColor = th.Color(t.accentColor, v)
	t.accent.Refresh()

	canvas.Refresh(t.container)
}
