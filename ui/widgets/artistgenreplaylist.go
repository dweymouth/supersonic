package widgets

import (
	"strconv"
	"supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ArtistGenrePlaylistItemModel struct {
	ID         string
	Name       string
	AlbumCount int
	TrackCount int
	Favorite   bool
}

type ArtistGenrePlaylist struct {
	widget.BaseWidget

	Items          []ArtistGenrePlaylistItemModel
	ShowAlbumCount bool
	ShowTrackCount bool
	OnNavTo        func(string)

	list *widget.List
}

type ArtistGenrePlaylistRow struct {
	widget.BaseWidget

	Item     ArtistGenrePlaylistItemModel
	OnTapped func()

	nameLabel       *widget.Label
	albumCountLabel *widget.Label
	trackCountLabel *widget.Label

	container *fyne.Container
}

func NewArtistGenrePlaylistRow() *ArtistGenrePlaylistRow {
	a := &ArtistGenrePlaylistRow{
		nameLabel:       widget.NewLabel(""),
		albumCountLabel: widget.NewLabel("alCount"),
		trackCountLabel: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.container = container.New(layouts.NewColumnsLayout([]float32{-1, 100, 100}), a.nameLabel, a.albumCountLabel, a.trackCountLabel)
	return a
}

func NewArtistGenrePlaylist(items []ArtistGenrePlaylistItemModel) *ArtistGenrePlaylist {
	a := &ArtistGenrePlaylist{
		Items: items,
	}
	a.ExtendBaseWidget(a)
	a.list = widget.NewList(
		func() int { return len(a.Items) },
		func() fyne.CanvasObject {
			r := NewArtistGenrePlaylistRow()
			r.OnTapped = func() { a.onRowDoubleTapped(r.Item) }
			return r
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*ArtistGenrePlaylistRow)
			row.Item = a.Items[id]
			row.albumCountLabel.Hidden = !a.ShowAlbumCount
			row.trackCountLabel.Hidden = !a.ShowTrackCount
			row.nameLabel.Text = row.Item.Name
			row.albumCountLabel.Text = strconv.Itoa(row.Item.AlbumCount)
			row.trackCountLabel.Text = strconv.Itoa(row.Item.TrackCount)
			row.Refresh()
		},
	)
	return a
}

func (a *ArtistGenrePlaylist) onRowDoubleTapped(item ArtistGenrePlaylistItemModel) {
	if a.OnNavTo != nil {
		a.OnNavTo(item.ID)
	}
}

func (a *ArtistGenrePlaylistRow) Tapped(*fyne.PointEvent) {
	if a.OnTapped != nil {
		a.OnTapped()
	}
}

func (a *ArtistGenrePlaylist) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.list)
}

func (a *ArtistGenrePlaylistRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
