package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type genrePageAdapter struct {
	genre string
	contr *controller.Controller
	mp    mediaprovider.MediaProvider
	pm    *backend.PlaybackManager
}

func NewGenrePage(genre string, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) Page {
	adapter := &genrePageAdapter{genre: genre, contr: contr, mp: mp, pm: pm}
	return NewGridViewPage(adapter, pool, mp, im)
}

func (g *genrePageAdapter) Title() string { return g.genre }

func (g *genrePageAdapter) Filter() *mediaprovider.AlbumFilter {
	return &mediaprovider.AlbumFilter{Genres: []string{g.genre}}
}

func (g *genrePageAdapter) PlaceholderResource() fyne.Resource {
	return myTheme.AlbumIcon
}

func (g *genrePageAdapter) Route() controller.Route {
	return controller.GenreRoute(g.genre)
}

func (g *genrePageAdapter) ActionButton() *widget.Button {
	fn := func() { go g.pm.PlayRandomSongs(g.genre) }
	return widget.NewButtonWithIcon(" Play random", myTheme.ShuffleIcon, fn)
}

func (a *genrePageAdapter) Iter(sortOrder string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.IterateAlbums(sortOrder, filter))
}

func (a *genrePageAdapter) SearchIter(query string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.SearchAlbums(query, filter))
}

func (g *genrePageAdapter) ConnectGridActions(gv *widgets.GridView) {
	g.contr.ConnectAlbumGridActions(gv)
}
