package dialogs

import (
	"strconv"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type AlbumFilterDialog struct {
	widget.BaseWidget

	OnDismissed func()

	filter    *backend.AlbumFilter
	container *fyne.Container
}

func NewAlbumFilterDialog(filter *backend.AlbumFilter) *AlbumFilterDialog {
	a := &AlbumFilterDialog{filter: filter}
	a.ExtendBaseWidget(a)

	// setup min and max year filters
	yearValidator := func(curText string, r rune) bool {
		return unicode.IsDigit(r) && len(curText) <= 3
	}
	minYear := widgets.NewTextRestrictedEntry(yearValidator)
	minYear.OnChanged = func(yearStr string) {
		if i, err := strconv.Atoi(yearStr); err == nil {
			a.filter.MinYear = i
		}
	}
	if filter.MinYear > 0 {
		minYear.Text = strconv.Itoa(filter.MinYear)
	}
	maxYear := widgets.NewTextRestrictedEntry(yearValidator)
	maxYear.OnChanged = func(yearStr string) {
		if i, err := strconv.Atoi(yearStr); err == nil {
			a.filter.MaxYear = i
		}
	}
	if filter.MaxYear > 0 {
		maxYear.Text = strconv.Itoa(filter.MaxYear)
	}

	var isNotFavorite *widget.Check
	// setup is favorite/not favorite filters
	isFavorite := widget.NewCheck("Is favorite", func(fav bool) {
		if fav {
			isNotFavorite.SetChecked(false)
		}
		a.filter.ExcludeUnfavorited = fav
	})
	isNotFavorite = widget.NewCheck("Is not favorite", func(fav bool) {
		if fav {
			isFavorite.SetChecked(false)
		}
		a.filter.ExcludeFavorited = fav
	})

	a.container = container.NewVBox(
		container.NewHBox(widget.NewLabel("Year from"), minYear, widget.NewLabel("to"), maxYear),
		isFavorite,
		isNotFavorite,
	)

	return a
}

func (a *AlbumFilterDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
