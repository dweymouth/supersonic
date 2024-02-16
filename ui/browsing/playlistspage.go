package browsing

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PlaylistsPage struct {
	widget.BaseWidget

	pool              *util.WidgetPool
	cfg               *backend.PlaylistsPageConfig
	contr             *controller.Controller
	mp                mediaprovider.MediaProvider
	playlists         []*mediaprovider.Playlist
	searchedPlaylists []*mediaprovider.Playlist

	viewToggle *widgets.ToggleButtonGroup
	searcher   *widgets.SearchEntry
	titleDisp  *widget.RichText
	container  *fyne.Container
	listView   *PlaylistList
	listSort   widgets.ListHeaderSort
	gridView   *widgets.GridView
}

func NewPlaylistsPage(contr *controller.Controller, pool *util.WidgetPool, cfg *backend.PlaylistsPageConfig, mp mediaprovider.MediaProvider) *PlaylistsPage {
	activeView := 0
	if cfg.InitialView == "Grid" {
		activeView = 1
	}
	return newPlaylistsPage(contr, pool, cfg, mp, "", activeView, widgets.ListHeaderSort{})
}

func newPlaylistsPage(contr *controller.Controller, pool *util.WidgetPool, cfg *backend.PlaylistsPageConfig, mp mediaprovider.MediaProvider, searchText string, activeView int, listSort widgets.ListHeaderSort) *PlaylistsPage {
	a := &PlaylistsPage{
		pool:      pool,
		cfg:       cfg,
		mp:        mp,
		contr:     contr,
		listSort:  listSort,
		titleDisp: widget.NewRichTextWithText("Playlists"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.searcher = widgets.NewSearchEntry()
	a.searcher.PlaceHolder = "Search page"
	a.searcher.OnSearched = a.onSearched
	a.searcher.Entry.Text = searchText
	a.viewToggle = widgets.NewToggleButtonGroup(0,
		widget.NewButtonWithIcon("", theme.NewThemedResource(res.ResListSvg), a.showListView),
		widget.NewButtonWithIcon("", theme.NewThemedResource(res.ResGridSvg), a.showGridView))
	a.viewToggle.SetActivatedButton(activeView)
	if activeView == 0 {
		a.createListView()
		a.buildContainer(a.listView)
	} else {
		a.createGridView(nil)
		a.buildContainer(a.gridView)
	}

	go a.load(searchText != "")
	return a
}

func (a *PlaylistsPage) load(searchOnLoad bool) {
	playlists, err := a.mp.GetPlaylists()
	if err != nil {
		log.Printf("error loading playlists: %v", err.Error())
	}
	a.playlists = playlists
	if searchOnLoad {
		a.onSearched(a.searcher.Entry.Text)
	} else {
		a.refreshView(playlists)
	}
}

func (a *PlaylistsPage) createListView() {
	a.listView = NewPlaylistList(a.listSort)
	a.listView.OnNavTo = a.showPlaylistPage
}

func (a *PlaylistsPage) createGridView(playlists []*mediaprovider.Playlist) {
	model := createPlaylistGridViewModel(playlists)
	if g := a.pool.Obtain(util.WidgetTypeGridView); g != nil {
		a.gridView = g.(*widgets.GridView)
		a.gridView.Placeholder = myTheme.PlaylistIcon
		a.gridView.ResetFixed(model)
	} else {
		a.gridView = widgets.NewFixedGridView(model, a.contr.App.ImageManager, myTheme.PlaylistIcon)
	}
	a.gridView.OnPlay = func(id string, shuffle bool) {
		go a.contr.App.PlaybackManager.PlayPlaylist(id, 0, shuffle)
	}
	a.gridView.OnAddToQueue = func(id string) {
		go a.contr.App.PlaybackManager.LoadPlaylist(id, true, false)
	}
	a.gridView.OnShowItemPage = a.showPlaylistPage
	a.gridView.OnShowSecondaryPage = nil
	a.gridView.OnAddToPlaylist = func(id string) {
		go func() {
			pl, err := a.contr.App.ServerManager.Server.GetPlaylist(id)
			if err != nil {
				log.Printf("error loading playlist: %s", err.Error())
				return
			}
			a.contr.DoAddTracksToPlaylistWorkflow(sharedutil.TracksToIDs(pl.Tracks))
		}()
	}
	a.gridView.OnDownload = func(id string) {
		go func() {
			pl, err := a.contr.App.ServerManager.Server.GetPlaylist(id)
			if err != nil {
				log.Printf("error loading playlist: %s", err.Error())
				return
			}
			a.contr.ShowDownloadDialog(pl.Tracks, pl.Name)
		}()
	}
}

func (a *PlaylistsPage) showListView() {
	a.cfg.InitialView = "List" // save setting
	if a.listView == nil {
		a.createListView()
		if a.searcher.Entry.Text != "" {
			a.listView.SetPlaylists(a.searchedPlaylists)
		} else {
			a.listView.SetPlaylists(a.playlists)
		}
	}
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.listView
	a.container.Objects[0].Refresh()
}

func (a *PlaylistsPage) showGridView() {
	a.cfg.InitialView = "Grid" // save setting
	if a.gridView == nil {
		playlists := a.playlists
		if a.searcher.Entry.Text != "" {
			playlists = a.searchedPlaylists
		}
		a.createGridView(playlists)
	}
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.gridView
	a.container.Objects[0].Refresh()
}

func createPlaylistGridViewModel(playlists []*mediaprovider.Playlist) []widgets.GridViewItemModel {
	return sharedutil.MapSlice(playlists, func(pl *mediaprovider.Playlist) widgets.GridViewItemModel {
		tracks := "tracks"
		if pl.TrackCount == 1 {
			tracks = "track"
		}
		return widgets.GridViewItemModel{
			Name:       pl.Name,
			ID:         pl.ID,
			CoverArtID: pl.CoverArtID,
			Secondary:  []string{fmt.Sprintf("%d %s", pl.TrackCount, tracks)},
		}
	})
}

func (a *PlaylistsPage) showPlaylistPage(id string) {
	a.contr.NavigateTo(controller.PlaylistRoute(id))
}

func (a *PlaylistsPage) onSearched(query string) {
	// since the playlist list is returned in full non-paginated, we will do our own
	// simple search based on the name, description, and owner, rather than calling a server API
	var playlists []*mediaprovider.Playlist
	if query == "" {
		a.searchedPlaylists = nil
		playlists = a.playlists
	} else {
		a.searchedPlaylists = sharedutil.FilterSlice(a.playlists, func(p *mediaprovider.Playlist) bool {
			qLower := strings.ToLower(query)
			return strings.Contains(strings.ToLower(p.Name), qLower) ||
				strings.Contains(strings.ToLower(p.Description), qLower) ||
				strings.Contains(strings.ToLower(p.Owner), qLower)
		})
		playlists = a.searchedPlaylists
	}
	a.refreshView(playlists)
}

// update the model for both views if initialized,
// refresh the active view
func (a *PlaylistsPage) refreshView(playlists []*mediaprovider.Playlist) {
	if a.listView != nil {
		a.listView.SetPlaylists(playlists)
	}
	if a.gridView != nil {
		a.gridView.ResetFixed(createPlaylistGridViewModel(playlists))
	}
	if a.viewToggle.ActivatedButtonIndex() == 0 {
		a.listView.Refresh()
	} else {
		a.gridView.Refresh()
	}
}

var _ Searchable = (*PlaylistsPage)(nil)

func (a *PlaylistsPage) SearchWidget() fyne.Focusable {
	return a.searcher
}

func (a *PlaylistsPage) Route() controller.Route {
	return controller.PlaylistsRoute()
}

func (a *PlaylistsPage) Reload() {
	go a.load(a.searcher.Entry.Text != "")
}

func (a *PlaylistsPage) Save() SavedPage {
	s := &savedPlaylistsPage{
		contr:      a.contr,
		pool:       a.pool,
		cfg:        a.cfg,
		mp:         a.mp,
		searchText: a.searcher.Entry.Text,
		activeView: a.viewToggle.ActivatedButtonIndex(),
	}
	if a.gridView != nil {
		a.gridView.Clear()
		a.pool.Release(util.WidgetTypeGridView, a.gridView)
	}
	if a.listView != nil {
		s.listSort = a.listView.sorting
	}
	return s
}

type savedPlaylistsPage struct {
	contr      *controller.Controller
	pool       *util.WidgetPool
	cfg        *backend.PlaylistsPageConfig
	mp         mediaprovider.MediaProvider
	searchText string
	activeView int
	listSort   widgets.ListHeaderSort
}

func (s *savedPlaylistsPage) Restore() Page {
	return newPlaylistsPage(s.contr, s.pool, s.cfg, s.mp, s.searchText, s.activeView, s.listSort)
}

func (a *PlaylistsPage) buildContainer(initialView fyne.CanvasObject) {
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(
			container.NewHBox(a.titleDisp, container.NewCenter(a.viewToggle), layout.NewSpacer(), searchVbox),
			nil, nil, nil, initialView))
}

func (a *PlaylistsPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type PlaylistList struct {
	widget.BaseWidget

	OnNavTo func(string)

	playlistsOrigOrder []*mediaprovider.Playlist
	playlists          []*mediaprovider.Playlist
	sorting            widgets.ListHeaderSort

	columnsLayout *layouts.ColumnsLayout
	header        *widgets.ListHeader
	list          *widgets.FocusList
	container     *fyne.Container
}

func NewPlaylistList(initialSort widgets.ListHeaderSort) *PlaylistList {
	a := &PlaylistList{
		sorting:       initialSort,
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, -1, 200, 125}),
	}
	a.buildHeader()
	a.list = widgets.NewFocusList(
		func() int {
			return len(a.playlists)
		},
		func() fyne.CanvasObject {
			r := NewPlaylistListRow(a.columnsLayout)
			r.OnTapped = func() { a.onRowTapped(r.ID) }
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*PlaylistListRow)
			row.ID = a.playlists[id].ID
			row.nameLabel.Text = a.playlists[id].Name
			row.descrptionLabel.Text = a.playlists[id].Description
			row.ownerLabel.Text = a.playlists[id].Owner
			row.trackCountLabel.Text = strconv.Itoa(a.playlists[id].TrackCount)
			row.Refresh()
		},
	)
	a.container = container.NewBorder(a.header, nil, nil, nil, a.list)
	a.ExtendBaseWidget(a)
	return a
}

func (p *PlaylistList) buildHeader() {
	p.header = widgets.NewListHeader([]widgets.ListColumn{
		{Text: "Name", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Description", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Owner", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Track Count", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false}}, p.columnsLayout)
	p.header.SetSorting(p.sorting)
	p.header.OnColumnSortChanged = p.onSorted
}

// Sets the playlists in the list. Does not issue Refresh call.
func (p *PlaylistList) SetPlaylists(playlists []*mediaprovider.Playlist) {
	p.playlistsOrigOrder = playlists
	p.doSortPlaylists()
}

func (p *PlaylistList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

func (p *PlaylistList) onRowTapped(id string) {
	if p.OnNavTo != nil {
		p.OnNavTo(id)
	}
}

func (p *PlaylistList) onSorted(sort widgets.ListHeaderSort) {
	p.sorting = sort
	p.doSortPlaylists()
	p.Refresh()
}

func (p *PlaylistList) doSortPlaylists() {
	if p.sorting.Type == widgets.SortNone {
		p.playlists = p.playlistsOrigOrder
		return
	}
	switch p.sorting.ColNumber {
	case 0: //Name
		p.stringSort(func(p *mediaprovider.Playlist) string { return p.Name })
	case 1: // Description
		p.stringSort(func(p *mediaprovider.Playlist) string { return p.Description })
	case 2: // Owner
		p.stringSort(func(p *mediaprovider.Playlist) string { return p.Owner })
	case 3: // Track Count
		p.intSort(func(p *mediaprovider.Playlist) int { return p.TrackCount })
	}
}

func (p *PlaylistList) stringSort(fieldFn func(*mediaprovider.Playlist) string) {
	new := make([]*mediaprovider.Playlist, len(p.playlistsOrigOrder))
	copy(new, p.playlistsOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		cmp := strings.Compare(fieldFn(new[i]), fieldFn(new[j]))
		if p.sorting.Type == widgets.SortDescending {
			return cmp > 0
		}
		return cmp < 0
	})
	p.playlists = new
}

func (p *PlaylistList) intSort(fieldFn func(*mediaprovider.Playlist) int) {
	new := make([]*mediaprovider.Playlist, len(p.playlistsOrigOrder))
	copy(new, p.playlistsOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		if p.sorting.Type == widgets.SortDescending {
			return fieldFn(new[i]) > fieldFn(new[j])
		}
		return fieldFn(new[i]) < fieldFn(new[j])
	})
	p.playlists = new
}

type PlaylistListRow struct {
	widget.BaseWidget

	ID       string
	OnTapped func()

	nameLabel       *widget.Label
	descrptionLabel *widget.Label
	ownerLabel      *widget.Label
	trackCountLabel *widget.Label

	container *fyne.Container
}

func NewPlaylistListRow(layout *layouts.ColumnsLayout) *PlaylistListRow {
	a := &PlaylistListRow{
		nameLabel:       widget.NewLabel(""),
		descrptionLabel: widget.NewLabel(""),
		ownerLabel:      widget.NewLabel(""),
		trackCountLabel: widget.NewLabel(""),
	}
	a.trackCountLabel.Alignment = fyne.TextAlignTrailing
	a.ownerLabel.Truncation = fyne.TextTruncateEllipsis
	a.container = container.New(layout, a.nameLabel, a.descrptionLabel, a.ownerLabel, a.trackCountLabel)
	a.ExtendBaseWidget(a)
	return a
}

func (a *PlaylistListRow) Tapped(*fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *PlaylistListRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
