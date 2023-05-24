package browsing

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type ArtistsPage struct {
	widget.BaseWidget

	contr               *controller.Controller
	im                  *backend.ImageManager
	pm                  *backend.PlaybackManager
	mp                  mediaprovider.MediaProvider
	artists             []*mediaprovider.Artist
	searchedArtists     []*mediaprovider.Artist
	grid                *widgets.GridView
	fullGridScrollPos   float32
	searchGridScrollPos float32
	searcher            *widgets.SearchEntry
	searchText          string
	titleDisp           *widget.RichText

	container *fyne.Container
}

func NewArtistsPage(
	contr *controller.Controller,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
) *ArtistsPage {
	return newArtistsPage(contr, pm, mp, im, "", 0, 0)
}

func newArtistsPage(
	contr *controller.Controller,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
	searchText string,
	fullGridScrollPos float32,
	searchGridScrollPos float32,
) *ArtistsPage {
	a := &ArtistsPage{
		contr:               contr,
		pm:                  pm,
		mp:                  mp,
		im:                  im,
		searchText:          searchText,
		fullGridScrollPos:   fullGridScrollPos,
		searchGridScrollPos: searchGridScrollPos,
	}
	a.ExtendBaseWidget(a)

	log.Printf("Scroll pos: full %0.2f search %0.2f", fullGridScrollPos, searchGridScrollPos)
	a.titleDisp = widget.NewRichTextWithText("Artists")
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.searcher = widgets.NewSearchEntry()
	a.searcher.OnSearched = func(query string) { a.onSearched(query, false /*firstLoad*/) }
	a.searcher.Entry.Text = searchText
	a.grid = widgets.NewFixedGridView(nil, a.im, myTheme.ArtistIcon)
	a.contr.ConnectArtistGridActions(a.grid)

	searchVbox := container.NewVBox(layout.NewSpacer(), a.searcher, layout.NewSpacer())
	a.container = container.NewBorder(
		container.NewHBox(util.NewHSpace(6),
			a.titleDisp, layout.NewSpacer(), searchVbox, util.NewHSpace(15)),
		nil, nil, nil, a.grid,
	)

	go a.load()
	return a
}

func (a *ArtistsPage) Reload() {
	a.searchGridScrollPos = 0
	a.fullGridScrollPos = 0
	go a.load()
}

func (a *ArtistsPage) load() {
	artists, err := a.mp.GetArtists()
	if err != nil {
		log.Printf("error loading artists: %v", err.Error())
	}
	a.artists = artists
	a.onSearched(a.searcher.Entry.Text, true)
}

func (a *ArtistsPage) onSearched(query string, firstLoad bool) {
	// since the playlist list is returned in full non-paginated, we will do our own
	// simple search based on the name, description, and owner, rather than calling a server API
	var artists []*mediaprovider.Artist
	scrollPos := float32(0)
	if query == "" {
		a.searchedArtists = nil
		artists = a.artists
		scrollPos = a.fullGridScrollPos
	} else {
		if firstLoad { // if reloading with a saved search state, set scroll position
			scrollPos = a.searchGridScrollPos
		}
		if a.searchText == "" {
			// if first search, capture scroll position of full, unsearched grid
			a.fullGridScrollPos = a.grid.GetScrollOffset()
		}
		qLower := strings.ToLower(query)
		a.searchedArtists = sharedutil.FilterSlice(a.artists, func(p *mediaprovider.Artist) bool {
			return strings.Contains(strings.ToLower(p.Name), qLower)
		})
		artists = a.searchedArtists
	}
	a.searchText = query
	a.grid.ResetFixed(createArtistsGridViewModel(artists))
	a.grid.Refresh()
	a.grid.ScrollToOffset(scrollPos)
}

func createArtistsGridViewModel(artists []*mediaprovider.Artist) []widgets.GridViewItemModel {
	return sharedutil.MapSlice(artists, func(ar *mediaprovider.Artist) widgets.GridViewItemModel {
		albums := "albums"
		if ar.AlbumCount == 1 {
			albums = "album"
		}
		return widgets.GridViewItemModel{
			Name:       ar.Name,
			ID:         ar.ID,
			CoverArtID: ar.CoverArtID,
			Secondary:  fmt.Sprintf("%d %s", ar.AlbumCount, albums),
		}
	})
}

func (a *ArtistsPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *ArtistsPage) Route() controller.Route {
	return controller.ArtistsRoute()
}

func (a *ArtistsPage) showArtistPage(id string) {
	a.contr.NavigateTo(controller.ArtistRoute(id))
}

func (a *ArtistsPage) Save() SavedPage {
	s := &savedArtistsPage{
		contr:             a.contr,
		im:                a.im,
		pm:                a.pm,
		mp:                a.mp,
		searchText:        a.searchText,
		fullGridScrollPos: a.fullGridScrollPos,
	}
	if a.searchText == "" {
		s.fullGridScrollPos = a.grid.GetScrollOffset()
	} else {
		s.searchGridScrollPos = a.grid.GetScrollOffset()
	}
	return s
}

type savedArtistsPage struct {
	contr               *controller.Controller
	im                  *backend.ImageManager
	pm                  *backend.PlaybackManager
	mp                  mediaprovider.MediaProvider
	searchText          string
	fullGridScrollPos   float32
	searchGridScrollPos float32
}

func (s *savedArtistsPage) Restore() Page {
	return newArtistsPage(s.contr, s.pm, s.mp, s.im, s.searchText, s.fullGridScrollPos, s.searchGridScrollPos)
}
