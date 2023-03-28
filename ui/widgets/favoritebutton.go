package widgets

import (
	"supersonic/res"
	"supersonic/ui/theme"

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
	var iconName fyne.ThemeIconName
	if f.IsFavorited {
		iconName = theme.IconNameFavorite
	} else {
		iconName = theme.IconNameNotFavorite
	}
	f.Icon = fyne.CurrentApp().Settings().Theme().Icon(iconName)
	f.Button.Refresh()
}
