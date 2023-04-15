package browsing

import (
	"log"
	"strings"
	"supersonic/backend"
	"supersonic/sharedutil"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type ArtistsGenresPage struct {
	widget.BaseWidget

	isGenresPage bool
	contr        *controller.Controller
	sm           *backend.ServerManager
	model        []widgets.ArtistGenreListItemModel
	list         *widgets.ArtistGenreList

	titleDisp *widget.RichText
	container *fyne.Container
	searcher  *widgets.Searcher
}

func NewArtistsGenresPage(isGenresPage bool, contr *controller.Controller, sm *backend.ServerManager) *ArtistsGenresPage {
	return newArtistsGenresPage(isGenresPage, contr, sm, "")
}

func newArtistsGenresPage(isGenresPage bool, contr *controller.Controller, sm *backend.ServerManager, searchText string) *ArtistsGenresPage {
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
	a.list = widgets.NewArtistGenreList(nil)
	a.list.ShowAlbumCount = true
	a.list.ShowTrackCount = isGenresPage
	a.list.OnNavTo = func(id string) {
		if a.isGenresPage {
			a.contr.NavigateTo(controller.GenreRoute(id))
		} else {
			a.contr.NavigateTo(controller.ArtistRoute(id))
		}
	}
	a.searcher = widgets.NewSearcher()
	a.searcher.OnSearched = a.onSearched
	a.searcher.Entry.Text = searchText
	a.buildContainer()
	go a.load(searchText != "")
	return a
}

// should be called asynchronously
func (a *ArtistsGenresPage) load(searchOnLoad bool) {
	if a.isGenresPage {
		genres, err := a.sm.Server.GetGenres()
		if err != nil {
			log.Printf("error loading genres: %v", err.Error())
		}
		a.model = a.buildGenresListModel(genres)
	} else {
		artists, err := a.sm.Server.GetArtists(nil)
		if err != nil {
			log.Printf("error loading artists: %v", err.Error())
		}
		a.model = a.buildArtistListModel(artists)
	}
	if searchOnLoad {
		a.onSearched(a.searcher.Entry.Text)
	} else {
		a.list.Items = a.model
		a.list.Refresh()
	}
}

func (a *ArtistsGenresPage) onSearched(query string) {
	// since the artists and genres lists are returned in full non-paginated, we will do our own
	// simple search based on the artist/genre name, rather than calling a server API
	if query == "" {
		a.list.Items = a.model
	} else {
		result := sharedutil.FilterSlice(a.model, func(x widgets.ArtistGenreListItemModel) bool {
			return strings.Contains(strings.ToLower(x.Name), strings.ToLower(query))
		})
		a.list.Items = result
	}
	a.list.Refresh()
}

var _ Searchable = (*ArtistsGenresPage)(nil)

func (a *ArtistsGenresPage) SearchWidget() fyne.Focusable {
	return a.searcher.Entry
}

func (a *ArtistsGenresPage) Route() controller.Route {
	if a.isGenresPage {
		return controller.GenresRoute()
	}
	return controller.ArtistsRoute()
}

func (a *ArtistsGenresPage) Reload() {
	go a.load(false)
}

func (a *ArtistsGenresPage) Save() SavedPage {
	return &savedArtistsGenresPage{
		isGenresPage: a.isGenresPage,
		contr:        a.contr,
		sm:           a.sm,
		searchText:   a.searcher.Entry.Text,
	}
}

type savedArtistsGenresPage struct {
	isGenresPage bool
	contr        *controller.Controller
	sm           *backend.ServerManager
	searchText   string
}

func (s *savedArtistsGenresPage) Restore() Page {
	return newArtistsGenresPage(s.isGenresPage, s.contr, s.sm, s.searchText)
}

func (a *ArtistsGenresPage) buildArtistListModel(artists *subsonic.ArtistsID3) []widgets.ArtistGenreListItemModel {
	model := make([]widgets.ArtistGenreListItemModel, 0)
	for _, idx := range artists.Index {
		for _, artist := range idx.Artist {
			model = append(model, widgets.ArtistGenreListItemModel{
				ID:         artist.ID,
				Name:       artist.Name,
				AlbumCount: artist.AlbumCount,
				Favorite:   artist.Starred != time.Time{},
			})
		}
	}
	return model
}

func (a *ArtistsGenresPage) buildGenresListModel(genres []*subsonic.Genre) []widgets.ArtistGenreListItemModel {
	model := make([]widgets.ArtistGenreListItemModel, 0)
	for _, genre := range genres {
		model = append(model, widgets.ArtistGenreListItemModel{
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
	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher.Entry, layout.NewSpacer())
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(
			container.New(&layouts.MaxPadLayout{PadLeft: -5},
				container.NewHBox(a.titleDisp, layout.NewSpacer(), searchVbox)),
			nil, nil, nil, a.list))
}

func (a *ArtistsGenresPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
