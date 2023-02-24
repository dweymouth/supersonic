package browsing

import (
	"log"
	"supersonic/backend"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type ArtistsGenresPage struct {
	widget.BaseWidget

	isGenresPage bool
	contr        controller.Controller
	sm           *backend.ServerManager
	titleDisp    *widget.RichText
	container    *fyne.Container
	list         *widgets.ArtistGenrePlaylist
}

func NewArtistsGenresPage(isGenresPage bool, contr controller.Controller, sm *backend.ServerManager) *ArtistsGenresPage {
	title := "Artists"
	if isGenresPage {
		title = "Genres"
	}
	a := &ArtistsGenresPage{
		isGenresPage: isGenresPage,
		contr:        contr,
		sm:           sm,
		titleDisp:    widget.NewRichTextWithText(title),
	}
	a.ExtendBaseWidget(a)
	a.titleDisp.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameHeadingText
	a.list = widgets.NewArtistGenrePlaylist(nil)
	a.list.ShowAlbumCount = true
	a.list.ShowTrackCount = isGenresPage
	a.list.OnNavTo = func(id string) {
		if a.isGenresPage {
			a.contr.NavigateTo(controller.GenreRoute(id))
		} else {
			a.contr.NavigateTo(controller.ArtistRoute(id))
		}
	}
	a.buildContainer()
	go a.load()
	return a
}

// should be called asynchronously
func (a *ArtistsGenresPage) load() {
	if a.isGenresPage {
		genres, err := a.sm.Server.GetGenres()
		if err != nil {
			log.Printf("error loading genres: %v", err.Error())
		}
		a.list.Items = a.buildGenresListModel(genres)
	} else {
		artists, err := a.sm.Server.GetArtists(nil)
		if err != nil {
			log.Printf("error loading artists: %v", err.Error())
		}
		a.list.Items = a.buildArtistListModel(artists)
	}
	a.Refresh()
}

func (a *ArtistsGenresPage) Route() controller.Route {
	if a.isGenresPage {
		return controller.GenresRoute()
	}
	return controller.ArtistsRoute()
}

func (a *ArtistsGenresPage) Reload() {
	go a.load()
}

func (a *ArtistsGenresPage) Save() SavedPage {
	return &savedArtistsGenresPage{
		isGenresPage: a.isGenresPage,
		contr:        a.contr,
		sm:           a.sm,
	}
}

type savedArtistsGenresPage struct {
	isGenresPage bool
	contr        controller.Controller
	sm           *backend.ServerManager
}

func (s *savedArtistsGenresPage) Restore() Page {
	return NewArtistsGenresPage(s.isGenresPage, s.contr, s.sm)
}

func (a *ArtistsGenresPage) buildArtistListModel(artists *subsonic.ArtistsID3) []widgets.ArtistGenrePlaylistItemModel {
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

func (a *ArtistsGenresPage) buildGenresListModel(genres []*subsonic.Genre) []widgets.ArtistGenrePlaylistItemModel {
	model := make([]widgets.ArtistGenrePlaylistItemModel, 0)
	for _, genre := range genres {
		model = append(model, widgets.ArtistGenrePlaylistItemModel{
			ID:         genre.Name,
			Name:       genre.Name,
			AlbumCount: genre.AlbumCount,
			TrackCount: genre.SongCount,
			Favorite:   false,
		})
	}
	return model
}

func (a *ArtistsGenresPage) buildContainer() {
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(a.titleDisp, nil, nil, nil, a.list))
}

func (a *ArtistsGenresPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
