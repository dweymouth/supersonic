package browsing

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type GenresPage struct {
	widget.BaseWidget

	contr  *controller.Controller
	mp     mediaprovider.MediaProvider
	genres []*mediaprovider.Genre
	list   *GenreList

	titleDisp *widget.RichText
	container *fyne.Container
	searcher  *widgets.SearchEntry
}

func NewGenresPage(contr *controller.Controller, mp mediaprovider.MediaProvider) *GenresPage {
	return newGenresPage(contr, mp, "", widgets.ListHeaderSort{})
}

func newGenresPage(contr *controller.Controller, mp mediaprovider.MediaProvider, searchText string, sorting widgets.ListHeaderSort) *GenresPage {
	a := &GenresPage{
		contr:     contr,
		mp:        mp,
		titleDisp: widget.NewRichTextWithText("Genres"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = NewGenreList(sorting)
	a.list.OnNavTo = func(id string) { a.contr.NavigateTo(controller.GenreRoute(id)) }
	a.searcher = widgets.NewSearchEntry()
	a.searcher.PlaceHolder = "Search page"
	a.searcher.OnSearched = a.onSearched
	a.searcher.Entry.Text = searchText
	a.buildContainer()
	go a.load(searchText != "")
	return a
}

// should be called asynchronously
func (a *GenresPage) load(searchOnLoad bool) {
	genres, err := a.mp.GetGenres()
	if err != nil {
		log.Printf("error loading genres: %v", err.Error())
	}
	a.genres = genres
	if searchOnLoad {
		a.onSearched(a.searcher.Entry.Text)
	} else {
		a.list.SetGenres(a.genres)
		a.list.Refresh()
	}
}

func (a *GenresPage) onSearched(query string) {
	// since the artists and genres lists are returned in full non-paginated, we will do our own
	// simple search based on the artist/genre name, rather than calling a server API
	if query == "" {
		a.list.SetGenres(a.genres)
	} else {
		query = strings.ToLower(query)
		result := sharedutil.FilterSlice(a.genres, func(x *mediaprovider.Genre) bool {
			return strings.Contains(strings.ToLower(x.Name), query)
		})
		a.list.SetGenres(result)
	}
	a.list.Refresh()
}

var _ Searchable = (*GenresPage)(nil)

func (a *GenresPage) SearchWidget() fyne.Focusable {
	return a.searcher
}

func (a *GenresPage) Route() controller.Route {
	return controller.GenresRoute()
}

func (a *GenresPage) Reload() {
	go a.load(false)
}

func (a *GenresPage) Save() SavedPage {
	return &savedGenresPage{
		contr:      a.contr,
		mp:         a.mp,
		searchText: a.searcher.Entry.Text,
		sorting:    a.list.sorting,
	}
}

type savedGenresPage struct {
	contr      *controller.Controller
	mp         mediaprovider.MediaProvider
	searchText string
	sorting    widgets.ListHeaderSort
}

func (s *savedGenresPage) Restore() Page {
	return newGenresPage(s.contr, s.mp, s.searchText, s.sorting)
}

func (a *GenresPage) buildContainer() {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(
			container.New(&layouts.MaxPadLayout{PadLeft: -5},
				container.NewHBox(a.titleDisp, layout.NewSpacer(), searchVbox)),
			nil, nil, nil, a.list))
}

func (a *GenresPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type GenreList struct {
	widget.BaseWidget

	OnNavTo func(string)

	sorting         widgets.ListHeaderSort
	genres          []*mediaprovider.Genre
	genresOrigOrder []*mediaprovider.Genre

	columnsLayout *layouts.ColumnsLayout
	hdr           *widgets.ListHeader
	list          *widgets.DisabledList
	container     *fyne.Container
}

type GenreListRow struct {
	widget.BaseWidget

	Item     *mediaprovider.Genre
	OnTapped func()

	nameLabel       *widget.Label
	albumCountLabel *widget.Label
	trackCountLabel *widget.Label

	container *fyne.Container
}

func NewGenreListRow(layout *layouts.ColumnsLayout) *GenreListRow {
	a := &GenreListRow{
		nameLabel:       widget.NewLabel(""),
		albumCountLabel: widget.NewLabel(""),
		trackCountLabel: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.albumCountLabel.Alignment = fyne.TextAlignTrailing
	a.trackCountLabel.Alignment = fyne.TextAlignTrailing
	a.container = container.New(layout, a.nameLabel, a.albumCountLabel, a.trackCountLabel)
	return a
}

func NewGenreList(sorting widgets.ListHeaderSort) *GenreList {
	a := &GenreList{
		sorting:       sorting,
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, 125, 125}),
	}
	a.ExtendBaseWidget(a)
	a.hdr = widgets.NewListHeader([]widgets.ListColumn{
		{"Name", fyne.TextAlignLeading, false}, {"Album Count", fyne.TextAlignTrailing, false}, {"Track Count", fyne.TextAlignTrailing, false}}, a.columnsLayout)
	a.hdr.SetSorting(sorting)
	a.hdr.OnColumnSortChanged = a.onSorted
	a.list = widgets.NewDisabledList(
		func() int { return len(a.genres) },
		func() fyne.CanvasObject {
			r := NewGenreListRow(a.columnsLayout)
			r.OnTapped = func() { a.onRowDoubleTapped(r.Item) }
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*GenreListRow)
			row.Item = a.genres[id]
			row.nameLabel.Text = row.Item.Name
			row.albumCountLabel.Text = strconv.Itoa(row.Item.AlbumCount)
			row.trackCountLabel.Text = strconv.Itoa(row.Item.TrackCount)
			row.Refresh()
		},
	)
	a.container = container.NewBorder(a.hdr, nil, nil, nil, a.list)
	return a
}

func (g *GenreList) SetGenres(genres []*mediaprovider.Genre) {
	g.genresOrigOrder = genres
	g.doSortGenres()
	g.Refresh()
}

func (g *GenreList) onSorted(sort widgets.ListHeaderSort) {
	g.sorting = sort
	g.doSortGenres()
	g.Refresh()
}

func (g *GenreList) doSortGenres() {
	if g.sorting.Type == widgets.SortNone {
		g.genres = g.genresOrigOrder
		return
	}
	switch g.sorting.ColNumber {
	case 0: //Name
		g.stringSort(func(g *mediaprovider.Genre) string { return g.Name })
	case 1: // Album Count
		g.intSort(func(g *mediaprovider.Genre) int { return g.AlbumCount })
	case 2: // Track Count
		g.intSort(func(g *mediaprovider.Genre) int { return g.TrackCount })
	}
}

func (g *GenreList) stringSort(fieldFn func(*mediaprovider.Genre) string) {
	new := make([]*mediaprovider.Genre, len(g.genresOrigOrder))
	copy(new, g.genresOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		cmp := strings.Compare(fieldFn(new[i]), fieldFn(new[j]))
		if g.sorting.Type == widgets.SortDescending {
			return cmp > 0
		}
		return cmp < 0
	})
	g.genres = new
}

func (g *GenreList) intSort(fieldFn func(*mediaprovider.Genre) int) {
	new := make([]*mediaprovider.Genre, len(g.genresOrigOrder))
	copy(new, g.genresOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		if g.sorting.Type == widgets.SortDescending {
			return fieldFn(new[i]) > fieldFn(new[j])
		}
		return fieldFn(new[i]) < fieldFn(new[j])
	})
	g.genres = new
}

func (a *GenreList) onRowDoubleTapped(item *mediaprovider.Genre) {
	if a.OnNavTo != nil {
		a.OnNavTo(item.Name)
	}
}

func (a *GenreListRow) Tapped(*fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *GenreList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *GenreListRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
