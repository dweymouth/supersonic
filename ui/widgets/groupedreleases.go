package widgets

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type GroupedReleasesModel struct {
	Albums       []*GridViewItemModel
	Compilations []*GridViewItemModel
	EPs          []*GridViewItemModel
	Singles      []*GridViewItemModel
}

type GroupedReleases struct {
	widget.BaseWidget

	Model GroupedReleasesModel

	OnPlay              func(id string, shuffle bool)
	OnPlayNext          func(id string)
	OnAddToQueue        func(id string)
	OnAddToPlaylist     func(id string)
	OnFavorite          func(id string, fav bool)
	OnDownload          func(id string)
	OnShare             func(id string)
	OnShowItemPage      func(id string)
	OnShowSecondaryPage func(id string)

	menu               *widget.PopUpMenu
	shareMenuItem      *fyne.MenuItem
	menuGridViewItemId string
	imageFetcher       util.ImageFetcher
	cardPool           sync.Pool

	content *fyne.Container
}

func NewGroupedReleases(model GroupedReleasesModel, fetch util.ImageFetcher) *GroupedReleases {
	g := &GroupedReleases{
		Model:        model,
		imageFetcher: fetch,
	}
	g.cardPool.New = func() any { return g.createNewItemCard() }
	g.ExtendBaseWidget(g)
	g.content = container.NewVBox()
	return g
}

func (g *GroupedReleases) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.content)
}

func (g *GroupedReleases) Refresh() {

}

func (g *GroupedReleases) createNewItemCard() fyne.CanvasObject {
	card := NewGridViewItem(myTheme.AlbumIcon)
	card.SetSize(backend.AppInstance().Config.GridView.CardSize)
	card.ItemIndex = -1
	card.ImgLoader = util.NewThumbnailLoader(g.imageFetcher, card.Cover.SetImage)
	card.ImgLoader.OnBeforeLoad = func() { card.Cover.SetImage(nil) }
	card.OnPlay = func() { g.onPlay(card.ItemID(), false) }
	card.OnFavorite = func(fav bool) {
		if g.OnFavorite != nil {
			g.OnFavorite(card.itemID, fav)
		}
	}
	card.OnShowSecondaryPage = func(id string) {
		if g.OnShowSecondaryPage != nil {
			g.OnShowSecondaryPage(id)
		}
	}
	card.OnShowItemPage = func() {
		if g.OnShowItemPage != nil {
			g.OnShowItemPage(card.ItemID())
		}
	}
	card.OnShowContextMenu = func(p fyne.Position) {
		g.showContextMenu(card, p)
	}
	return card
}

func (g *GroupedReleases) showContextMenu(card *GridViewItem, pos fyne.Position) {
	g.menuGridViewItemId = card.ItemID()
	if g.menu == nil {
		play := fyne.NewMenuItem(lang.L("Play"), func() { g.onPlay(g.menuGridViewItemId, false) })
		play.Icon = theme.MediaPlayIcon()
		shuffle := fyne.NewMenuItem(lang.L("Shuffle"), func() { g.onPlay(g.menuGridViewItemId, true) })
		shuffle.Icon = myTheme.ShuffleIcon
		queueNext := fyne.NewMenuItem(lang.L("Play next"), func() {
			if g.OnPlayNext != nil {
				g.OnPlayNext(g.menuGridViewItemId)
			}
		})
		queueNext.Icon = myTheme.PlayNextIcon
		queue := fyne.NewMenuItem(lang.L("Add to queue"), func() {
			if g.OnAddToQueue != nil {
				g.OnAddToQueue(g.menuGridViewItemId)
			}
		})
		queue.Icon = theme.ContentAddIcon()
		playlist := fyne.NewMenuItem(lang.L("Add to playlist")+"...", func() {
			if g.OnAddToPlaylist != nil {
				g.OnAddToPlaylist(g.menuGridViewItemId)
			}
		})
		playlist.Icon = myTheme.PlaylistIcon
		download := fyne.NewMenuItem(lang.L("Download")+"...", func() {
			if g.OnDownload != nil {
				g.OnDownload(g.menuGridViewItemId)
			}
		})
		download.Icon = theme.DownloadIcon()
		g.shareMenuItem = fyne.NewMenuItem(lang.L("Share")+"...", func() {
			g.OnShare(g.menuGridViewItemId)
		})
		g.shareMenuItem.Icon = myTheme.ShareIcon
		g.menu = widget.NewPopUpMenu(fyne.NewMenu("", play, shuffle, queueNext, queue, playlist, download, g.shareMenuItem),
			fyne.CurrentApp().Driver().CanvasForObject(g))
	}
	g.menu.ShowAtPosition(pos)
}

func (g *GroupedReleases) onPlay(itemID string, shuffle bool) {
	if g.OnPlay != nil {
		g.OnPlay(itemID, shuffle)
	}
}
