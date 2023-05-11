package widgets

import (
	"strconv"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type AlbumFilterButton struct {
	widget.Button

	OnChanged func()

	filter *backend.AlbumFilter
	dialog *widget.PopUp
}

func NewAlbumFilterButton(filter *backend.AlbumFilter) *AlbumFilterButton {
	a := &AlbumFilterButton{
		filter: filter,
		Button: widget.Button{
			Icon: theme.AlbumIcon,
		},
	}
	a.OnTapped = a.showFilterDialog
	a.ExtendBaseWidget(a)
	return a
}

func (a *AlbumFilterButton) Refresh() {
	if a.filter.IsEmpty() {
		a.Importance = widget.MediumImportance
	} else {
		a.Importance = widget.HighImportance
	}
	a.Button.Refresh()
}

func (a *AlbumFilterButton) onFilterChanged() {
	a.Refresh()
	if a.OnChanged != nil {
		a.OnChanged()
	}
}

func (a *AlbumFilterButton) showFilterDialog() {
	if a.dialog == nil {
		filterDlg := NewAlbumFilterPopup(a.filter)
		filterDlg.OnChanged = a.onFilterChanged
		a.dialog = widget.NewPopUp(filterDlg, fyne.CurrentApp().Driver().CanvasForObject(a))
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a)
	a.dialog.ShowAtPosition(fyne.NewPos(pos.X+a.Size().Width/2-a.dialog.MinSize().Width/2, pos.Y+a.Size().Height))
}

type AlbumFilterPopup struct {
	widget.BaseWidget

	OnChanged func()

	filter    *backend.AlbumFilter
	container *fyne.Container
}

func NewAlbumFilterPopup(filter *backend.AlbumFilter) *AlbumFilterPopup {
	a := &AlbumFilterPopup{filter: filter}
	a.ExtendBaseWidget(a)

	debounceOnChanged := util.NewDebouncer(350*time.Millisecond, a.emitOnChanged)

	// setup min and max year filters
	yearValidator := func(curText, selText string, r rune) bool {
		return unicode.IsDigit(r) && (len(selText) > 0 || len(curText) <= 3)
	}
	minYear := NewTextRestrictedEntry(yearValidator)
	minYear.SetMinCharWidth(4)
	minYear.OnChanged = func(yearStr string) {
		if yearStr == "" {
			a.filter.MinYear = 0
		} else if i, err := strconv.Atoi(yearStr); err == nil {
			a.filter.MinYear = i
		}
		debounceOnChanged()
	}
	if filter.MinYear > 0 {
		minYear.Text = strconv.Itoa(filter.MinYear)
	}
	maxYear := NewTextRestrictedEntry(yearValidator)
	maxYear.SetMinCharWidth(4)
	maxYear.OnChanged = func(yearStr string) {
		if yearStr == "" {
			a.filter.MaxYear = 0
		} else if i, err := strconv.Atoi(yearStr); err == nil {
			a.filter.MaxYear = i
		}
		debounceOnChanged()
	}
	if filter.MaxYear > 0 {
		maxYear.Text = strconv.Itoa(filter.MaxYear)
	}

	// setup is favorite/not favorite filters
	var isNotFavorite *widget.Check
	isFavorite := widget.NewCheck("Is favorite", func(fav bool) {
		if fav {
			isNotFavorite.SetChecked(false)
		}
		a.filter.ExcludeUnfavorited = fav
		debounceOnChanged()
	})
	isNotFavorite = widget.NewCheck("Is not favorite", func(fav bool) {
		if fav {
			isFavorite.SetChecked(false)
		}
		a.filter.ExcludeFavorited = fav
		debounceOnChanged()
	})

	// setup container
	title := widget.NewLabel("Album filters")
	title.TextStyle.Bold = true
	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()),
		container.NewHBox(widget.NewLabel("Year from"), minYear, widget.NewLabel("to"), maxYear),
		container.NewHBox(isFavorite, isNotFavorite),
	)

	return a
}

func (a *AlbumFilterPopup) emitOnChanged() {
	if a.OnChanged != nil {
		a.OnChanged()
	}
}

func (a *AlbumFilterPopup) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
