package browsing

import (
	"log"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"
)

type albumsPageAdapter struct {
	cfg       *backend.AlbumsPageConfig
	contr     *controller.Controller
	mp        mediaprovider.MediaProvider
	pm        *backend.PlaybackManager
	filter    mediaprovider.AlbumFilter
	filterBtn *widgets.AlbumFilterButton
}

func NewAlbumsPage(cfg *backend.AlbumsPageConfig, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) Page {
	adapter := &albumsPageAdapter{cfg: cfg, contr: contr, mp: mp, pm: pm}
	return NewGridViewPage(adapter, pool, mp, im)
}

func (a *albumsPageAdapter) Title() string { return lang.L("Albums") }

func (a *albumsPageAdapter) Filter() mediaprovider.AlbumFilter {
	if a.filter == nil {
		a.filter = mediaprovider.NewAlbumFilter(
			mediaprovider.AlbumFilterOptions{},
		)
	}
	return a.filter
}

func (a *albumsPageAdapter) FilterButton() widgets.FilterButton[mediaprovider.Album, mediaprovider.AlbumFilterOptions] {
	if a.filterBtn == nil {
		a.filterBtn = widgets.NewAlbumFilterButton(a.Filter(), a.mp.GetGenres)
	}
	return a.filterBtn
}

func (a *albumsPageAdapter) PlaceholderResource() fyne.Resource { return myTheme.AlbumIcon }

func (a *albumsPageAdapter) Route() controller.Route { return controller.AlbumsRoute() }

func (a *albumsPageAdapter) SortOrders() ([]string, int) {
	orders := a.mp.AlbumSortOrders()
	sortOrder := slices.Index(orders, a.cfg.SortOrder)
	if sortOrder < 0 {
		sortOrder = 0
	}

	return util.LocalizeSlice(orders), sortOrder
}

func (a *albumsPageAdapter) SaveSortOrder(orderIdx int) {
	a.cfg.SortOrder = a.mp.AlbumSortOrders()[orderIdx]
}

func (a *albumsPageAdapter) ActionButton() fyne.CanvasObject {
	fn := func() {
		go func() {
			if err := a.pm.PlayRandomAlbums(""); err != nil {
				log.Printf("error playing random albums: %v", err)
				fyne.Do(func() {
					a.contr.ToastProvider.ShowErrorToast(lang.L("Unable to play random albums"))
				})
			}
		}()
	}

	return widget.NewButtonWithIcon(lang.L("Play random"), myTheme.ShuffleIcon, fn)
}

func (a *albumsPageAdapter) Iter(sortOrderIdx int, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	sortOrder := a.mp.AlbumSortOrders()[sortOrderIdx]
	return widgets.NewGridViewAlbumIterator(a.mp.IterateAlbums(sortOrder, filter))
}

func (a *albumsPageAdapter) SearchIter(query string, filter mediaprovider.AlbumFilter) widgets.GridViewIterator {
	return widgets.NewGridViewAlbumIterator(a.mp.SearchAlbums(query, filter))
}

func (a *albumsPageAdapter) InitGrid(gv *widgets.GridView) {
	a.contr.ConnectAlbumGridActions(gv)
	gv.ShowSuffix = a.cfg.ShowYears
}

func (a *albumsPageAdapter) RefreshGrid(gv *widgets.GridView) {
	gv.ShowSuffix = a.cfg.ShowYears
	gv.Refresh()
}
