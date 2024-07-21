package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type genrePageAdapter struct {
	genre     string
	cfg       *backend.AlbumsPageConfig
	contr     *controller.Controller
	mp        mediaprovider.MediaProvider
	pm        *backend.PlaybackManager
	filter    mediaprovider.AlbumFilter
	filterBtn *widgets.AlbumFilterButton
}

func NewGenrePage(genre string, cfg *backend.AlbumsPageConfig, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) Page {
	adapter := &genrePageAdapter{genre: genre, cfg: cfg, contr: contr, mp: mp, pm: pm}
	return NewGridViewPage(adapter, pool, mp, im)
}

func (g *genrePageAdapter) Title() string { return g.genre }

func (g *genrePageAdapter) Filter() mediaprovider.AlbumFilter {
	if g.filter == nil {
		g.filter = mediaprovider.NewAlbumFilter(
			mediaprovider.AlbumFilterOptions{
				Genres: []string{g.genre},
			},
		)
	}
	return g.filter
}

func (g *genrePageAdapter) FilterButton() widgets.FilterButton[mediaprovider.Album, mediaprovider.AlbumFilterOptions] {
	if g.filterBtn == nil {
		g.filterBtn = widgets.NewAlbumFilterButton(g.Filter(), func() ([]*mediaprovider.Genre, error) { return nil, nil })
		g.filterBtn.GenreDisabled = true
	}
	return g.filterBtn
}

func (g *genrePageAdapter) PlaceholderResource() fyne.Resource {
	return myTheme.AlbumIcon
}

func (g *genrePageAdapter) Route() controller.Route {
	return controller.GenreRoute(g.genre)
}

func (g *genrePageAdapter) ActionButton() *widget.Button {
	fn := func() { go g.pm.PlayRandomSongs(g.genre) }
	return widget.NewButtonWithIcon(lang.L("Play random"), myTheme.ShuffleIcon, fn)
}

func (a *genrePageAdapter) Iter(sortOrderIdx int, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.IterateAlbums("", filter))
}

func (a *genrePageAdapter) SearchIter(query string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.SearchAlbums(query, filter))
}

func (g *genrePageAdapter) InitGrid(gv *widgets.GridView) {
	g.contr.ConnectAlbumGridActions(gv)
	gv.ShowSuffix = g.cfg.ShowYears
}

func (g *genrePageAdapter) RefreshGrid(gv *widgets.GridView) {
	gv.ShowSuffix = g.cfg.ShowYears
	gv.Refresh()
}
