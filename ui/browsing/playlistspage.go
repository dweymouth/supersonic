package browsing

import (
	"log"
	"strconv"
	"supersonic/backend"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type PlaylistsPage struct {
	widget.BaseWidget

	contr     *controller.Controller
	sm        *backend.ServerManager
	titleDisp *widget.RichText
	container *fyne.Container
	list      *PlaylistList
}

func NewPlaylistsPage(contr *controller.Controller, sm *backend.ServerManager) *PlaylistsPage {
	a := &PlaylistsPage{
		sm:        sm,
		contr:     contr,
		titleDisp: widget.NewRichTextWithText("Playlists"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = NewPlaylistList()
	a.list.OnNavTo = func(id string) {
		a.contr.NavigateTo(controller.PlaylistRoute(id))
	}
	a.buildContainer()
	go a.loadAsync()
	return a
}

func (a *PlaylistsPage) loadAsync() {
	playlists, err := a.sm.Server.GetPlaylists(nil)
	if err != nil {
		log.Printf("error loading playlists: %v", err.Error())
	}
	a.list.Playlists = playlists
	a.list.Refresh()
}

func (a *PlaylistsPage) Route() controller.Route {
	return controller.PlaylistsRoute()
}

func (a *PlaylistsPage) Reload() {
	go a.loadAsync()
}

func (a *PlaylistsPage) Save() SavedPage {
	return &savedPlaylistsPage{
		contr: a.contr,
		sm:    a.sm,
	}
}

type savedPlaylistsPage struct {
	contr *controller.Controller
	sm    *backend.ServerManager
}

func (s *savedPlaylistsPage) Restore() Page {
	return NewPlaylistsPage(s.contr, s.sm)
}

func (a *PlaylistsPage) buildContainer() {
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(a.titleDisp, nil, nil, nil, a.list))
}

func (a *PlaylistsPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type PlaylistList struct {
	widget.BaseWidget

	Playlists []*subsonic.Playlist
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
			row.descrptionLabel.Text = a.Playlists[id].Comment
			row.ownerLabel.Text = a.Playlists[id].Owner
			row.trackCountLabel.Text = strconv.Itoa(a.Playlists[id].SongCount)
			row.Refresh()
		},
	)
	a.container = container.NewBorder(a.header, nil, nil, nil, a.list)
	a.ExtendBaseWidget(a)
	return a
}

func (p *PlaylistList) buildHeader() {
	p.header = widgets.NewListHeader([]widgets.ListColumn{
		{"Name", false, false},
		{"Description", false, false},
		{"Owner", false, false},
		{"Track Count", true, false}}, p.columnsLayout)

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
