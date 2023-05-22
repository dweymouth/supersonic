package widgets

import (
	"strconv"

	"github.com/dweymouth/supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ArtistGenreListItemModel struct {
	ID         string
	Name       string
	AlbumCount int
	TrackCount int
	Favorite   bool
}

type ArtistGenreList struct {
	widget.BaseWidget

	Items          []ArtistGenreListItemModel
	ShowAlbumCount bool
	ShowTrackCount bool
	OnNavTo        func(string)

	columnsLayout *layouts.ColumnsLayout
	hdr           *ListHeader
	list          *widget.List
	container     *fyne.Container
}

type ArtistGenreListRow struct {
	widget.BaseWidget

	Item     ArtistGenreListItemModel
	OnTapped func()

	nameLabel       *widget.Label
	albumCountLabel *widget.Label
	trackCountLabel *widget.Label

	container *fyne.Container
}

func NewArtistGenreListRow(layout *layouts.ColumnsLayout) *ArtistGenreListRow {
	a := &ArtistGenreListRow{
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

func NewArtistGenreList(items []ArtistGenreListItemModel) *ArtistGenreList {
	a := &ArtistGenreList{
		Items:         items,
		columnsLayout: layouts.NewColumnsLayout([]float32{-1, 125, 125}),
	}
	a.ExtendBaseWidget(a)
	a.hdr = NewListHeader([]ListColumn{
		{"Name", false, false}, {"Album Count", true, false}, {"Track Count", true, false}}, a.columnsLayout)
	a.hdr.DisableSorting = true
	a.list = widget.NewList(
		func() int { return len(a.Items) },
		func() fyne.CanvasObject {
			r := NewArtistGenreListRow(a.columnsLayout)
			r.OnTapped = func() { a.onRowDoubleTapped(r.Item) }
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*ArtistGenreListRow)
			row.Item = a.Items[id]
			row.albumCountLabel.Hidden = !a.ShowAlbumCount
			row.trackCountLabel.Hidden = !a.ShowTrackCount
			row.nameLabel.Text = row.Item.Name
			row.albumCountLabel.Text = strconv.Itoa(row.Item.AlbumCount)
			row.trackCountLabel.Text = strconv.Itoa(row.Item.TrackCount)
			row.Refresh()
		},
	)
	a.container = container.NewBorder(a.hdr, nil, nil, nil, a.list)
	return a
}

func (a *ArtistGenreList) Refresh() {
	a.hdr.SetColumnVisible(1, a.ShowAlbumCount)
	a.hdr.SetColumnVisible(2, a.ShowTrackCount)
	a.BaseWidget.Refresh()
}

func (a *ArtistGenreList) onRowDoubleTapped(item ArtistGenreListItemModel) {
	if a.OnNavTo != nil {
		a.OnNavTo(item.ID)
	}
}

func (a *ArtistGenreListRow) Tapped(*fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *ArtistGenreList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *ArtistGenreListRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
