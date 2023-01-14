package browsing

import (
	"log"
	"supersonic/backend"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type ArtistsPage struct {
	widget.BaseWidget

	sm        *backend.ServerManager
	nav       func(Route)
	titleDisp *widget.RichText
	container *fyne.Container
	list      *widgets.ArtistGenrePlaylist
}

func NewArtistsPage(sm *backend.ServerManager, nav func(Route)) *ArtistsPage {
	a := &ArtistsPage{
		sm:        sm,
		nav:       nav,
		titleDisp: widget.NewRichTextWithText("Artists"),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = widgets.NewArtistGenrePlaylist(nil)
	a.list.ShowAlbumCount = true
	a.list.ShowTrackCount = false
	a.buildContainer()
	return a
}

func (a *ArtistsPage) loadAsync() {
	artists, err := a.sm.Server.GetArtists(nil)
	if err != nil {
		log.Printf("error loading artists: %v", err.Error())
	}
	a.list.Items = a.buildListModel(artists)
	a.list.Refresh()
}

func (a *ArtistsPage) buildListModel(artists *subsonic.ArtistsID3) []widgets.ArtistGenrePlaylistItemModel {
	model := make([]widgets.ArtistGenrePlaylistItemModel, 0)
	for _, idx := range artists.Index {
		for _, artist := range idx.Artist {
			model = append(model, widgets.ArtistGenrePlaylistItemModel{
				ID:         artist.ID,
				Name:       artist.Name,
				AlbumCount: artist.AlbumCount,
				Favorite:   artist.Starred != time.Time{},
			})
		}
	}
	return model
}

func (a *ArtistsPage) buildContainer() {
	a.container = container.NewBorder(a.titleDisp, nil, nil, nil, a.list)
}

func (a *ArtistsPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
