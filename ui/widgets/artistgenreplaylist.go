package widgets

import (
	"fyne.io/fyne/v2"
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

	container *fyne.Container
}

func NewArtistGenrePlaylistRow() *ArtistGenrePlaylistRow {
	a := &ArtistGenrePlaylistRow{}
	a.ExtendBaseWidget(a)
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
