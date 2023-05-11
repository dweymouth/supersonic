package widgets

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type FavoriteButton struct {
	widget.Button

	IsFavorited bool
}

func NewFavoriteButton(onTapped func()) *FavoriteButton {
	f := &FavoriteButton{
		Button: widget.Button{
			OnTapped: onTapped,
			Icon:     res.ResHeartOutlineInvertPng,
		},
	}
	f.ExtendBaseWidget(f)
	return f
}

func (f *FavoriteButton) Tapped(e *fyne.PointEvent) {
	f.IsFavorited = !f.IsFavorited
	f.Button.Tapped(e)
	f.Refresh()
}

func (f *FavoriteButton) Refresh() {
	if f.IsFavorited {
		f.Icon = theme.FavoriteIcon
	} else {
		f.Icon = theme.NotFavoriteIcon
	}
	f.Button.Refresh()
}

type AlbumFilterButton struct {
	widget.Button

	filter *backend.AlbumFilter

	dialogConstructor func(*backend.AlbumFilter) fyne.CanvasObject
	dialog            *widget.PopUp
}

func NewAlbumFilterButton(filter *backend.AlbumFilter, dialogConstructor func(*backend.AlbumFilter) fyne.CanvasObject) *AlbumFilterButton {
	a := &AlbumFilterButton{
		filter:            filter,
		dialogConstructor: dialogConstructor,
		Button: widget.Button{
			Icon: theme.AlbumIcon,
		},
	}
	a.OnTapped = a.showFilterDialog
	a.ExtendBaseWidget(a)
	return a
}

func (a *AlbumFilterButton) showFilterDialog() {
	if a.dialog == nil {
		a.dialog = widget.NewModalPopUp(a.dialogConstructor(a.filter), fyne.CurrentApp().Driver().CanvasForObject(a))
	}
	a.dialog.Show()
}
