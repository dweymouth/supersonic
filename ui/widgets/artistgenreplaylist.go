package widgets

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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

	list *widget.List
}

type ArtistGenrePlaylistRow struct {
	widget.BaseWidget

	Item ArtistGenrePlaylistItemModel

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
	// layouts.NewColumnsLayout([]float32{-1, 50, 50})
	a.container = container.New(layout.NewHBoxLayout(), a.nameLabel, a.albumCountLabel, a.trackCountLabel)
	return a
}

func NewArtistGenrePlaylist(items []ArtistGenrePlaylistItemModel) *ArtistGenrePlaylist {
	a := &ArtistGenrePlaylist{
		Items: items,
	}
	a.ExtendBaseWidget(a)
	a.list = widget.NewList(
		func() int { return len(a.Items) },
		func() fyne.CanvasObject { return NewArtistGenrePlaylistRow() },
		func(id widget.ListItemID, item fyne.CanvasObject) {
			row := item.(*ArtistGenrePlaylistRow)
			row.Item = a.Items[id]
			row.albumCountLabel.Hidden = a.ShowAlbumCount
			row.trackCountLabel.Hidden = a.ShowTrackCount
			row.nameLabel.Text = row.Item.Name
			row.albumCountLabel.Text = strconv.Itoa(row.Item.AlbumCount)
			row.trackCountLabel.Text = strconv.Itoa(row.Item.TrackCount)
			row.Refresh()
		},
	)
	return a
}

func (a *ArtistGenrePlaylist) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.list)
}

func (a *ArtistGenrePlaylistRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
