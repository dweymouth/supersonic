package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	myTheme "github.com/dweymouth/supersonic/ui/theme"
)

type IconButtonSize int

const (
	IconButtonSizeNormal IconButtonSize = iota
	IconButtonSizeBigger
	IconButtonSizeSmaller
)

type IconButton struct {
	widget.BaseWidget

	Highlighted bool
	IconSize    IconButtonSize
	OnTapped    func()

	icon    fyne.Resource
	focused bool
	hovered bool

	themed *theme.ThemedResource
	img    *canvas.Image
}

var (
	_ fyne.Tappable     = (*IconButton)(nil)
	_ fyne.Focusable    = (*IconButton)(nil)
	_ desktop.Hoverable = (*IconButton)(nil)
)

func NewIconButton(icon fyne.Resource, onTapped func()) *IconButton {
	i := &IconButton{icon: icon, OnTapped: onTapped}
	i.ExtendBaseWidget(i)
	return i
}

func (i *IconButton) SetIcon(icon fyne.Resource) {
	i.icon = icon
	if i.img != nil {
		i.themed = theme.NewThemedResource(icon)
		i.img.Resource = i.themed
		i.Refresh()
	}
}

func (i *IconButton) Tapped(*fyne.PointEvent) {
	if i.OnTapped != nil {
		i.OnTapped()
	}
}

func (i *IconButton) FocusGained() {
	if !i.focused {
		defer i.Refresh()
	}
	i.focused = true
}

func (i *IconButton) FocusLost() {
	if i.focused {
		defer i.Refresh()
	}
	i.focused = false
}

func (i *IconButton) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeySpace {
		i.Tapped(nil)
	}
}

func (i *IconButton) TypedRune(r rune) {
}

func (i *IconButton) MouseIn(*desktop.MouseEvent) {
	if !i.hovered {
		defer i.Refresh()
	}
	i.hovered = true
}

func (i *IconButton) MouseOut() {
	if i.hovered {
		defer i.Refresh()
	}
	i.hovered = false
}

func (i *IconButton) MouseMoved(*desktop.MouseEvent) {
}

func (i *IconButton) MinSize() fyne.Size {
	return i.iconSize()
}

func (i *IconButton) iconSize() fyne.Size {
	switch i.IconSize {
	case IconButtonSizeBigger:
		return fyne.NewSquareSize(theme.IconInlineSize() * 2)
	case IconButtonSizeSmaller:
		return fyne.NewSquareSize(theme.IconInlineSize())
	default:
		return fyne.NewSquareSize(theme.IconInlineSize() * 1.3333)
	}
}

func (i *IconButton) Refresh() {
	if i.img == nil {
		return
	}
	if i.Highlighted || i.focused {
		i.themed.ColorName = theme.ColorNamePrimary
	} else if i.hovered {
		i.themed.ColorName = myTheme.ColorNameHoveredIconButton
	} else {
		i.themed.ColorName = myTheme.ColorNameIconButton
	}
	i.img.SetMinSize(i.iconSize())
	i.img.Refresh()
}

func (i *IconButton) CreateRenderer() fyne.WidgetRenderer {
	if i.img == nil {
		i.themed = theme.NewThemedResource(i.icon)
		i.img = canvas.NewImageFromResource(i.themed)
		i.img.FillMode = canvas.ImageFillContain
		i.img.SetMinSize(i.iconSize())
	}
	return widget.NewSimpleRenderer(container.NewCenter(i.img))
}
