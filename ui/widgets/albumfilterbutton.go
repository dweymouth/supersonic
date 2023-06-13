package widgets

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type AlbumFilterButton struct {
	widget.Button

	OnChanged        func()
	GenreDisabled    bool
	FavoriteDisabled bool

	genreListChan chan []string

	filter *mediaprovider.AlbumFilter
	dialog *widget.PopUp
}

func NewAlbumFilterButton(filter *mediaprovider.AlbumFilter, fetchGenresFunc func() ([]*mediaprovider.Genre, error)) *AlbumFilterButton {
	a := &AlbumFilterButton{
		filter: filter,
		Button: widget.Button{
			Icon: myTheme.FilterIcon,
		},
	}
	a.OnTapped = a.showFilterDialog
	a.ExtendBaseWidget(a)
	a.genreListChan = make(chan []string)
	go func() {
		if genres, err := fetchGenresFunc(); err == nil {
			genreNames := sharedutil.MapSlice(genres, func(g *mediaprovider.Genre) string {
				return g.Name
			})
			sort.Strings(genreNames)
			a.genreListChan <- genreNames
		}
	}()
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
	genreFilter   *GenreFilterSubsection
	filterBtn     *AlbumFilterButton
	container     *fyne.Container
}

func NewAlbumFilterPopup(filter *AlbumFilterButton) *AlbumFilterPopup {
	a := &AlbumFilterPopup{filterBtn: filter}
	a.ExtendBaseWidget(a)

	debounceOnChanged := util.NewDebouncer(350*time.Millisecond, a.emitOnChanged)

	// setup min and max year filters
	yearValidator := func(curText, selText string, r rune) bool {
		l := len(curText) - len(selText)
		return unicode.IsDigit(r) && l <= 3 && (l > 0 || r != '0')
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

	// create genre filter subsection
	a.genreFilter = NewGenreFilterSubsection(func(selectedGenres []string) {
		a.filterBtn.filter.Genres = selectedGenres
		debounceOnChanged()
	}, a.filterBtn.filter.Genres)
	a.genreFilter.Hidden = a.filterBtn.GenreDisabled

	// setup container
	title := widget.NewLabel("Album filters")
	title.TextStyle.Bold = true
	a.container = container.NewVBox(
		container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer()),
		container.NewHBox(widget.NewLabel("Year from"), minYear, widget.NewLabel("to"), maxYear),
		container.NewHBox(a.isFavorite, a.isNotFavorite),
		a.genreFilter,
	)

	go func() {
		a.genreFilter.SetGenreList(<-a.filterBtn.genreListChan)
	}()

	return a
}

func (a *AlbumFilterPopup) Tapped(_ *fyne.PointEvent) {
	// swallow the Tapped event so that the popup is
	// only dismissed by clicking outside of it
}

func (a *AlbumFilterPopup) Refresh() {
	a.isFavorite.Hidden = a.filterBtn.FavoriteDisabled
	a.isNotFavorite.Hidden = a.filterBtn.FavoriteDisabled
	a.genreFilter.Hidden = a.filterBtn.GenreDisabled
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

type GenreFilterSubsection struct {
	widget.BaseWidget

	genreList []string
	onChanged func([]string)

	selectedGenres      map[string]interface{}
	selectedGenresMutex sync.RWMutex

	filterText         *widget.Entry
	numSelectedText    *widget.Label
	allBtn             *widget.Button
	noneBtn            *widget.Button
	listModelMutex     sync.RWMutex
	genreListViewModel []string
	genreListView      *widget.List

	container *fyne.Container
}

func NewGenreFilterSubsection(onChanged func([]string), initialSelectedGenres []string) *GenreFilterSubsection {
	g := &GenreFilterSubsection{
		onChanged:      onChanged,
		selectedGenres: make(map[string]interface{}),
	}
	g.ExtendBaseWidget(g)

	for _, genre := range initialSelectedGenres {
		g.selectedGenres[genre] = nil
	}

	g.genreListView = widget.NewList(
		func() int {
			g.listModelMutex.RLock()
			defer g.listModelMutex.RUnlock()
			return len(g.genreListViewModel)
		},
		func() fyne.CanvasObject {
			return newGenreListViewRow(g.onGenreSelected)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			g.listModelMutex.RLock()
			defer g.listModelMutex.RUnlock()
			genre := g.genreListViewModel[id]
			g.selectedGenresMutex.RLock()
			_, selected := g.selectedGenres[genre]
			g.selectedGenresMutex.RUnlock()
			row := obj.(*genreListViewRow)
			row.ID = id
			row.Text = genre
			row.Checked = selected
			row.Refresh()
		},
	)
	g.filterText = widget.NewEntry()
	g.filterText.SetPlaceHolder("Filter genres")
	i := NewTappableIcon(theme.ContentClearIcon())
	i.NoPointerCursor = true
	i.OnTapped = func() { g.filterText.SetText("") }
	g.filterText.ActionItem = i
	debouncer := util.NewDebouncer(300*time.Millisecond, g.updateGenreListView)
	g.filterText.OnChanged = func(_ string) {
		debouncer()
	}
	g.allBtn = widget.NewButton("All", func() { g.selectAllOrNoneInView(false) })
	g.noneBtn = widget.NewButton("None", func() { g.selectAllOrNoneInView(true) })

	title := widget.NewRichTextWithText("Genres")
	title.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = true
	g.numSelectedText = widget.NewLabel("(none selected)")
	g.updateNumSelectedText()
	titleRow := container.New(&layouts.HboxCustomPadding{ExtraPad: -10}, title, g.numSelectedText)

	filterRow := container.NewBorder(nil, nil, nil, container.NewHBox(g.allBtn, g.noneBtn), g.filterText)
	g.container = container.NewBorder(titleRow, nil, nil, nil,
		container.New(&layouts.MaxPadLayout{PadLeft: 5, PadRight: 5},
			container.NewBorder(filterRow, nil, nil, nil, g.genreListView),
		),
	)
	return g
}

func (g *GenreFilterSubsection) MinSize() fyne.Size {
	return fyne.NewSize(g.BaseWidget.MinSize().Width, 250)
}

func (g *GenreFilterSubsection) SetGenreList(genres []string) {
	g.genreList = genres
	g.updateGenreListView()
}

func (g *GenreFilterSubsection) updateGenreListView() {
	g.listModelMutex.Lock()
	if g.filterText.Text == "" {
		g.genreListViewModel = g.genreList
	} else {
		filterText := strings.ToLower(g.filterText.Text)
		g.genreListViewModel = sharedutil.FilterSlice(g.genreList, func(genre string) bool {
			return strings.Contains(strings.ToLower(genre), filterText)
		})
	}
	g.listModelMutex.Unlock()
	g.genreListView.Refresh()
}

func (g *GenreFilterSubsection) onGenreSelected(row widget.ListItemID, selected bool) {
	g.listModelMutex.RLock()
	g.selectedGenresMutex.Lock()
	if selected {
		g.selectedGenres[g.genreListViewModel[row]] = nil
	} else {
		delete(g.selectedGenres, g.genreListViewModel[row])
	}
	g.selectedGenresMutex.Unlock()
	g.listModelMutex.RUnlock()
	g.invokeOnChanged()
}

func (g *GenreFilterSubsection) selectAllOrNoneInView(none bool) {
	g.listModelMutex.RLock()
	g.selectedGenresMutex.Lock()
	for _, genre := range g.genreListViewModel {
		if none {
			delete(g.selectedGenres, genre)
		} else {
			g.selectedGenres[genre] = nil
		}
	}
	g.selectedGenresMutex.Unlock()
	g.listModelMutex.RUnlock()
	g.genreListView.Refresh()
	g.invokeOnChanged()
}

func (g *GenreFilterSubsection) invokeOnChanged() {
	g.selectedGenresMutex.RLock()
	g.updateNumSelectedText()
	genres := make([]string, 0, len(g.selectedGenres))
	for genre := range g.selectedGenres {
		genres = append(genres, genre)
	}
	g.selectedGenresMutex.RUnlock()
	g.onChanged(genres)
}

func (g *GenreFilterSubsection) updateNumSelectedText() {
	numText := "none"
	if l := len(g.selectedGenres); l > 0 {
		numText = strconv.Itoa(l)
	}
	g.numSelectedText.SetText(fmt.Sprintf("(%s selected)", numText))
}

func (g *GenreFilterSubsection) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.container)
}

type genreListViewRow struct {
	widget.Check

	ID widget.ListItemID
}

func newGenreListViewRow(onChanged func(widget.ListItemID, bool)) *genreListViewRow {
	g := &genreListViewRow{}
	g.ExtendBaseWidget(g)
	g.OnChanged = func(b bool) {
		onChanged(g.ID, b)
	}
	return g
}
