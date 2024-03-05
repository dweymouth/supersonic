package browsing

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type albumsPageAdapter struct {
	cfg   *backend.AlbumsPageConfig
	contr *controller.Controller
	mp    mediaprovider.MediaProvider
	pm    *backend.PlaybackManager
}

func NewAlbumsPage(cfg *backend.AlbumsPageConfig, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) Page {
	adapter := &albumsPageAdapter{cfg: cfg, contr: contr, mp: mp, pm: pm}
	return NewGridViewPage(adapter, pool, mp, im)
}

func (a *albumsPageAdapter) Title() string { return "Albums" }

func (a *albumsPageAdapter) Filter() *mediaprovider.AlbumFilter {
	return &mediaprovider.AlbumFilter{}
}

func (a *albumsPageAdapter) PlaceholderResource() fyne.Resource { return myTheme.AlbumIcon }

func (a *albumsPageAdapter) Route() controller.Route { return controller.AlbumsRoute() }

func (a *albumsPageAdapter) SortOrders() ([]string, string) {
	orders := a.mp.AlbumSortOrders()
	sortOrder := a.cfg.SortOrder
	if !slices.Contains(orders, sortOrder) {
		sortOrder = string(orders[0])
	}
	return orders, sortOrder
}

func (a *albumsPageAdapter) SaveSortOrder(order string) {
	a.cfg.SortOrder = order
}

func (a *albumsPageAdapter) ActionButton() *widget.Button { return nil }

func (a *albumsPageAdapter) Iter(sortOrder string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.IterateAlbums(sortOrder, filter))
}

func (a *albumsPageAdapter) SearchIter(query string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.SearchAlbums(query, filter))
}

func (a *albumsPageAdapter) ConnectGridActions(gv *widgets.GridView) {
	a.contr.ConnectAlbumGridActions(gv)
}
