package browsing

import (
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

type artistsPageAdapter struct {
	cfg    *backend.ArtistsPageConfig
	contr  *controller.Controller
	mp     mediaprovider.MediaProvider
	pm     *backend.PlaybackManager
	filter mediaprovider.ArtistFilter
}

func NewArtistsPage(cfg *backend.ArtistsPageConfig, pool *util.WidgetPool, contr *controller.Controller, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager) Page {
	adapter := &artistsPageAdapter{cfg: cfg, contr: contr, mp: mp, pm: pm}
	return NewGridViewPage(adapter, pool, mp, im)
}

func (a *artistsPageAdapter) Title() string { return lang.L("Artists") }

func (a *artistsPageAdapter) Filter() mediaprovider.ArtistFilter {
	if a.filter == nil {
		a.filter = mediaprovider.NewArtistFilter(
			mediaprovider.ArtistFilterOptions{},
		)
	}
	return a.filter
}

func (a *artistsPageAdapter) FilterButton() widgets.FilterButton[mediaprovider.Artist, mediaprovider.ArtistFilterOptions] {
	return nil
}

func (a *artistsPageAdapter) PlaceholderResource() fyne.Resource { return myTheme.ArtistIcon }

func (a *artistsPageAdapter) Route() controller.Route { return controller.ArtistsRoute() }

func (a *artistsPageAdapter) SortOrders() ([]string, string) {
	orders := a.mp.ArtistSortOrders()
	sortOrder := a.cfg.SortOrder
	if !slices.Contains(orders, sortOrder) {
		sortOrder = string(orders[0])
	}
	return orders, sortOrder
}

func (a *artistsPageAdapter) SaveSortOrder(order string) {
	a.cfg.SortOrder = order
}

func (a *artistsPageAdapter) ActionButton() *widget.Button { return nil }

func (a *artistsPageAdapter) Iter(sortOrder string, filter mediaprovider.ArtistFilter) widgets.GridViewIterator {
	return widgets.NewGridViewArtistIterator(a.mp.IterateArtists(sortOrder, filter))
}

func (a *artistsPageAdapter) SearchIter(query string, filter mediaprovider.ArtistFilter) widgets.GridViewIterator {
	return widgets.NewGridViewArtistIterator(a.mp.SearchArtists(query, filter))
}

func (a *artistsPageAdapter) InitGrid(gv *widgets.GridView) {
	canShareArtists := false
	if r, canShare := a.mp.(mediaprovider.SupportsSharing); canShare {
		canShareArtists = r.CanShareArtists()
	}
	gv.DisableSharing = !canShareArtists
	a.contr.ConnectArtistGridActions(gv)
}

func (a *artistsPageAdapter) RefreshGrid(gv *widgets.GridView) {
	gv.Refresh()
}
