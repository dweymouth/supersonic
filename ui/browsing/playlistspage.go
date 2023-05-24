package browsing

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PlaylistsPage struct {
	widget.BaseWidget

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
	gridView   *widgets.GridView
}

func NewPlaylistsPage(contr *controller.Controller, cfg *backend.PlaylistsPageConfig, mp mediaprovider.MediaProvider) *PlaylistsPage {
	activeView := 0
	if cfg.InitialView == "Grid" {
		activeView = 1
	}
	return newPlaylistsPage(contr, cfg, mp, "", activeView)
}

func newPlaylistsPage(contr *controller.Controller, cfg *backend.PlaylistsPageConfig, mp mediaprovider.MediaProvider, searchText string, activeView int) *PlaylistsPage {
	a := &PlaylistsPage{
		cfg:       cfg,
		mp:        mp,
		contr:     contr,
		titleDisp: widget.NewRichTextWithText("Playlists"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.searcher = widgets.NewSearchEntry()
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
	a.listView = NewPlaylistList()
	a.listView.OnNavTo = a.showPlaylistPage
}

func (a *PlaylistsPage) createGridView(playlists []*mediaprovider.Playlist) {
	model := createPlaylistGridViewModel(playlists)
	a.gridView = widgets.NewFixedGridView(model, a.contr.App.ImageManager, myTheme.PlaylistIcon)
	a.gridView.OnPlay = func(id string, shuffle bool) {
		go a.contr.App.PlaybackManager.PlayPlaylist(id, 0, shuffle)
	}
	a.gridView.OnAddToQueue = func(id string) {
		go a.contr.App.PlaybackManager.LoadPlaylist(id, true, false)
	}
	a.gridView.OnShowItemPage = a.showPlaylistPage
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
}

func (a *PlaylistsPage) showListView() {
	a.cfg.InitialView = "List" // save setting
	if a.listView == nil {
		a.createListView()
		if a.searcher.Entry.Text != "" {
			a.listView.Playlists = a.searchedPlaylists
		} else {
			a.listView.Playlists = a.playlists
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
			Secondary:  fmt.Sprintf("%d %s", pl.TrackCount, tracks),
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
		a.listView.Playlists = playlists
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
	return &savedPlaylistsPage{
		contr:      a.contr,
		cfg:        a.cfg,
		mp:         a.mp,
		searchText: a.searcher.Entry.Text,
		activeView: a.viewToggle.ActivatedButtonIndex(),
	}
}

type savedPlaylistsPage struct {
	contr      *controller.Controller
	cfg        *backend.PlaylistsPageConfig
	mp         mediaprovider.MediaProvider
	searchText string
	activeView int
}

func (s *savedPlaylistsPage) Restore() Page {
	return newPlaylistsPage(s.contr, s.cfg, s.mp, s.searchText, s.activeView)
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

	Playlists []*mediaprovider.Playlist
	OnNavTo   func(string)

	columnsLayout *layouts.ColumnsLayout
	header        *widgets.ListHeader
	list          *widget.List
	container     *fyne.Container
}

func NewPlaylistList() *PlaylistList {
	a := &PlaylistList{
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, -1, 200, 125}),
	}
	a.buildHeader()
	a.list = widget.NewList(
		func() int {
			return len(a.Playlists)
		},
		func() fyne.CanvasObject {
			r := NewPlaylistListRow(a.columnsLayout)
			r.OnTapped = func() { a.onRowTapped(r.ID) }
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*PlaylistListRow)
			row.ID = a.Playlists[id].ID
			row.nameLabel.Text = a.Playlists[id].Name
			row.descrptionLabel.Text = a.Playlists[id].Description
			row.ownerLabel.Text = a.Playlists[id].Owner
			row.trackCountLabel.Text = strconv.Itoa(a.Playlists[id].TrackCount)
			row.Refresh()
		},
	)
	a.container = container.NewBorder(a.header, nil, nil, nil, a.list)
	a.ExtendBaseWidget(a)
	return a
}

func (p *PlaylistList) buildHeader() {
	p.header = widgets.NewListHeader([]widgets.ListColumn{
		{"Name", fyne.TextAlignLeading, false},
		{"Description", fyne.TextAlignLeading, false},
		{"Owner", fyne.TextAlignLeading, false},
		{"Track Count", fyne.TextAlignTrailing, false}}, p.columnsLayout)
	p.header.DisableSorting = true

}

func (p *PlaylistList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

func (p *PlaylistList) onRowTapped(id string) {
	if p.OnNavTo != nil {
		p.OnNavTo(id)
	}
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
	a.ownerLabel.Wrapping = fyne.TextTruncate
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
