package widgets

import (
	"strconv"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type AlbumFilterButton struct {
	widget.Button

	OnChanged        func()
	GenreDisabled    bool
	FavoriteDisabled bool

	filter *mediaprovider.AlbumFilter
	dialog *widget.PopUp
}

func NewAlbumFilterButton(filter *mediaprovider.AlbumFilter) *AlbumFilterButton {
	a := &AlbumFilterButton{
		filter: filter,
		Button: widget.Button{
			Icon: theme.FilterIcon,
		},
	}
	a.OnTapped = a.showFilterDialog
	a.ExtendBaseWidget(a)
	return a
}

func (a *AlbumFilterButton) Refresh() {
	if a.filterEmpty() {
		a.Importance = widget.MediumImportance
	} else {
		a.Importance = widget.HighImportance
	}
	a.Button.Refresh()
}

func (a *AlbumFilterButton) filterEmpty() bool {
	return a.filter.MinYear == 0 && a.filter.MaxYear == 0 &&
		(a.FavoriteDisabled || !a.filter.ExcludeFavorited && !a.filter.ExcludeUnfavorited) &&
		(a.GenreDisabled || len(a.filter.Genres) == 0)
}

func (a *AlbumFilterButton) onFilterChanged() {
	a.Refresh()
	if a.OnChanged != nil {
		a.OnChanged()
	}
}

func (a *AlbumFilterButton) showFilterDialog() {
	if a.dialog == nil {
		filterDlg := NewAlbumFilterPopup(a)
		filterDlg.OnChanged = a.onFilterChanged
		a.dialog = widget.NewPopUp(filterDlg, fyne.CurrentApp().Driver().CanvasForObject(a))
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a)
	a.dialog.ShowAtPosition(fyne.NewPos(pos.X+a.Size().Width/2-a.dialog.MinSize().Width/2, pos.Y+a.Size().Height))
}

type AlbumFilterPopup struct {
	widget.BaseWidget

	OnChanged func()

	isFavorite    *widget.Check
	isNotFavorite *widget.Check
	filterBtn     *AlbumFilterButton
	container     *fyne.Container
}

func NewAlbumFilterPopup(filter *AlbumFilterButton) *AlbumFilterPopup {
	a := &AlbumFilterPopup{filterBtn: filter}
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
			a.filterBtn.filter.MinYear = 0
		} else if i, err := strconv.Atoi(yearStr); err == nil {
			a.filterBtn.filter.MinYear = i
		}
		debounceOnChanged()
	}
	if a.filterBtn.filter.MinYear > 0 {
		minYear.Text = strconv.Itoa(a.filterBtn.filter.MinYear)
	}
	maxYear := NewTextRestrictedEntry(yearValidator)
	maxYear.SetMinCharWidth(4)
	maxYear.OnChanged = func(yearStr string) {
		if yearStr == "" {
			a.filterBtn.filter.MaxYear = 0
		} else if i, err := strconv.Atoi(yearStr); err == nil {
			a.filterBtn.filter.MaxYear = i
		}
		debounceOnChanged()
	}
	if a.filterBtn.filter.MaxYear > 0 {
		maxYear.Text = strconv.Itoa(a.filterBtn.filter.MaxYear)
	}

	// setup is favorite/not favorite filters
	a.isFavorite = widget.NewCheck("Is favorite", func(fav bool) {
		if fav {
			a.isNotFavorite.SetChecked(false)
		}
		a.filterBtn.filter.ExcludeUnfavorited = fav
		debounceOnChanged()
	})
	a.isFavorite.Hidden = a.filterBtn.FavoriteDisabled
	a.isNotFavorite = widget.NewCheck("Is not favorite", func(fav bool) {
		if fav {
			a.isFavorite.SetChecked(false)
		}
		a.filterBtn.filter.ExcludeFavorited = fav
		debounceOnChanged()
	})
	a.isNotFavorite.Hidden = a.filterBtn.FavoriteDisabled

	// setup container
	title := widget.NewLabel("Album filters")
	title.TextStyle.Bold = true
	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()),
		container.NewHBox(widget.NewLabel("Year from"), minYear, widget.NewLabel("to"), maxYear),
		container.NewHBox(a.isFavorite, a.isNotFavorite),
	)

	return a
}

func (a *AlbumFilterPopup) Refresh() {
	a.isFavorite.Hidden = a.filterBtn.FavoriteDisabled
	a.isNotFavorite.Hidden = a.filterBtn.FavoriteDisabled
	a.BaseWidget.Refresh()
}

func (a *AlbumFilterPopup) emitOnChanged() {
	if a.OnChanged != nil {
		a.OnChanged()
	}
}

func (a *AlbumFilterPopup) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
