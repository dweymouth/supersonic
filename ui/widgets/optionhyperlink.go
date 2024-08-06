package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// OptionHyperlink is a widget that displays a "vertical dots"/"more" button
// to the right of the text, which can show a menu.
type OptionHyperlink struct {
	widget.BaseWidget
	h *ttwidget.Hyperlink
	b *IconButton

	OnShowMenu func(btnPos fyne.Position)

	layout optionHyperlinkLayout
}

func NewOptionHyperlink() *OptionHyperlink {
	o := &OptionHyperlink{
		h: ttwidget.NewHyperlink("", nil),
	}
	o.h.Truncation = fyne.TextTruncateEllipsis
	o.b = NewIconButton(theme.MoreVerticalIcon(), func() {
		if o.OnShowMenu != nil {
			o.OnShowMenu(fyne.CurrentApp().Driver().AbsolutePositionForObject(o.b))
		}
	})
	o.b.IconSize = IconButtonSizeSmaller
	o.ExtendBaseWidget(o)
	return o
}

func (o *OptionHyperlink) SetTextAndToolTip(text string) {
	o.h.SetText(text)
	o.h.SetToolTip(text)
	o.updatePreferredWidth()
	o.BaseWidget.Refresh()
}

func (o *OptionHyperlink) Text() string {
	return o.h.Text
}

func (o *OptionHyperlink) SetTextStyle(style fyne.TextStyle) {
	o.h.TextStyle = style
	o.updatePreferredWidth()
	o.BaseWidget.Refresh()
}

func (o *OptionHyperlink) SetOnTapped(f func()) {
	o.h.OnTapped = f
}

func (o *OptionHyperlink) SetMenuBtnEnabled(enabled bool) {
	o.b.Hidden = !enabled
}

func (o *OptionHyperlink) updatePreferredWidth() {
	var h widget.Hyperlink
	h.Text = o.Text()
	h.TextStyle = o.h.TextStyle
	o.layout.preferredWidth = h.MinSize().Width + theme.Padding()
}

func (o *OptionHyperlink) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(
		container.New(&o.layout, o.h, o.b),
	)
}

var _ fyne.Layout = (*optionHyperlinkLayout)(nil)

type optionHyperlinkLayout struct {
	preferredWidth float32
}

func (o *optionHyperlinkLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var minSize fyne.Size
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}
		ms := obj.MinSize()
		minSize.Height = fyne.Max(ms.Height, minSize.Height)
		minSize.Width += ms.Width
	}
	return minSize
}

// only supports the two objects
func (o *optionHyperlinkLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if !objects[1].Visible() {
		objects[0].Move(fyne.NewPos(0, 0))
		objects[0].Resize(size)
		return
	}
	btnWidth := objects[1].MinSize().Width
	hypWidth := fyne.Min(o.preferredWidth, size.Width-btnWidth)
	objects[0].Resize(fyne.NewSize(hypWidth, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(btnWidth, size.Height))
	objects[1].Move(fyne.NewPos(hypWidth-theme.Padding()*2, 0))
}
