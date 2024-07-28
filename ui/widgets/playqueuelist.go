package widgets

import (
	"image"
	"slices"
	"strconv"
	"sync"

	"fyne.io/fyne/v2/lang"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

const playQueueListThumbnailSize = 52

type PlayQueueListModel struct {
	Item     mediaprovider.MediaItem
	Selected bool
}

type PlayQueueList struct {
	widget.BaseWidget

	Reorderable    bool
	DisableRating  bool
	DisableSharing bool

	// user action callbacks
	OnPlayItemAt        func(idx int)
	OnPlaySelection     func(items []mediaprovider.MediaItem, shuffle bool)
	OnPlaySelectionNext func(items []mediaprovider.MediaItem)
	OnAddToQueue        func(items []mediaprovider.MediaItem)
	OnAddToPlaylist     func(trackIDs []string)
	OnSetFavorite       func(trackIDs []string, fav bool)
	OnSetRating         func(trackIDs []string, rating int)
	OnRemoveFromQueue   func(idxs []int)
	OnDownload          func(tracks []*mediaprovider.Track, downloadName string)
	OnShowTrackInfo     func(track *mediaprovider.Track)
	OnShare             func(tracks []*mediaprovider.Track)
	OnShowArtistPage    func(artistID string)
	OnReorderItems      func(idxs []int, reorderTo int)

	useNonQueueMenu bool
	menu            *widget.PopUpMenu // ctx menu for when only tracks are selected
	radiosMenu      *widget.PopUpMenu // ctx menu for when selection contains radios
	ratingSubmenu   *fyne.MenuItem
	infoMenuItem    *fyne.MenuItem
	shareMenuItem   *fyne.MenuItem

	nowPlayingID string

	list        *FocusList
	colLayout   *layouts.ColumnsLayout
	tracksMutex sync.RWMutex
	items       []*util.TrackListModel
}

func NewPlayQueueList(im *backend.ImageManager, useNonQueueMenu bool) *PlayQueueList {
	p := &PlayQueueList{useNonQueueMenu: useNonQueueMenu}
	p.ExtendBaseWidget(p)

	// #, Cover, Title/Artist, Time
	coverWidth := NewPlayQueueListRow(p, im, layout.NewSpacer()).cover.MinSize().Width
	p.colLayout = layouts.NewColumnsLayout([]float32{40, coverWidth, -1, 60})

	playIconResource := theme.NewThemedResource(theme.MediaPlayIcon())
	playIconResource.ColorName = theme.ColorNamePrimary
	playIconImg := canvas.NewImageFromResource(playIconResource)
	playIconImg.FillMode = canvas.ImageFillContain
	playIconImg.SetMinSize(fyne.NewSquareSize(theme.IconInlineSize() * 1.5))

	playingIcon := container.NewCenter(playIconImg)

	p.list = NewFocusList(
		p.lenTracks,
		func() fyne.CanvasObject {
			return NewPlayQueueListRow(p, im, playingIcon)
		},
		func(itemID widget.ListItemID, item fyne.CanvasObject) {
			p.tracksMutex.RLock()
			// we could have removed tracks from the list in between
			// Fyne calling the length callback and this update callback
			// so the itemID may be out of bounds. if so, do nothing.
			if itemID >= len(p.items) {
				p.tracksMutex.RUnlock()
				return
			}
			model := p.items[itemID]
			p.tracksMutex.RUnlock()

			tr := item.(*PlayQueueListRow)
			if tr.trackID != model.Item.Metadata().ID || tr.ListItemID != itemID {
				tr.ListItemID = itemID
			}
			tr.Update(model, itemID+1)
		},
	)
	p.list.OnDragBegin = func(id int) {
		if !p.items[id].Selected {
			p.selectTrack(id)
			p.list.Refresh()
		}
	}
	p.list.OnDragEnd = func(dragged, insertPos int) {
		if p.OnReorderItems != nil {
			p.OnReorderItems(p.selectedIdxs(), insertPos)
		}
	}

	return p
}

func (p *PlayQueueList) SetTracks(trs []*mediaprovider.Track) {
	p.tracksMutex.Lock()
	p.items = util.ToTrackListModels(trs)
	p.tracksMutex.Unlock()
	p.Refresh()
}

func (p *PlayQueueList) SetItems(items []mediaprovider.MediaItem) {
	p.tracksMutex.Lock()
	p.items = sharedutil.MapSlice(items, func(item mediaprovider.MediaItem) *util.TrackListModel {
		return &util.TrackListModel{Item: item}
	})
	p.tracksMutex.Unlock()
	p.Refresh()
}

func (p *PlayQueueList) Items() []mediaprovider.MediaItem {
	return sharedutil.MapSlice(p.items, func(item *util.TrackListModel) mediaprovider.MediaItem {
		return item.Item
	})
}

// Sets the currently playing item ID and updates the list rendering
func (p *PlayQueueList) SetNowPlaying(itemID string) {
	prevNowPlaying := p.nowPlayingID
	p.tracksMutex.RLock()
	trPrev, idxPrev := util.FindItemByID(p.items, prevNowPlaying)
	tr, idx := util.FindItemByID(p.items, itemID)
	p.tracksMutex.RUnlock()
	p.nowPlayingID = itemID
	if trPrev != nil {
		p.list.RefreshItem(idxPrev)
	}
	if tr != nil {
		p.list.RefreshItem(idx)
	}
}

func (p *PlayQueueList) SelectAll() {
	p.tracksMutex.RLock()
	util.SelectAllItems(p.items)
	p.tracksMutex.RUnlock()
	p.list.Refresh()
}

func (p *PlayQueueList) UnselectAll() {
	p.tracksMutex.RLock()
	util.UnselectAllItems(p.items)
	p.tracksMutex.RUnlock()
	p.list.Refresh()
}

func (p *PlayQueueList) Scroll(amount float32) {
	p.list.ScrollToOffset(p.list.GetScrollOffset() + amount)
}

func (p *PlayQueueList) ScrollToNowPlaying() {
	idx := slices.IndexFunc(p.items, func(item *util.TrackListModel) bool {
		return item.Item.Metadata().ID == p.nowPlayingID
	})
	if idx > 0 {
		p.list.ScrollTo(idx)
	}
}

func (p *PlayQueueList) Refresh() {
	p.list.EnableDragging = p.Reorderable
	p.BaseWidget.Refresh()
}

func (p *PlayQueueList) lenTracks() int {
	p.tracksMutex.RLock()
	defer p.tracksMutex.RUnlock()
	return len(p.items)
}

func (t *PlayQueueList) onArtistTapped(artistID string) {
	if t.OnShowArtistPage != nil {
		t.OnShowArtistPage(artistID)
	}
}

func (p *PlayQueueList) onPlayTrackAt(idx int) {
	if p.OnPlayItemAt != nil {
		p.OnPlayItemAt(idx)
	}
}

func (p *PlayQueueList) onSelectTrack(idx int) {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		mod := d.CurrentKeyModifiers()
		if mod&fyne.KeyModifierShortcutDefault != 0 {
			p.selectAddOrRemove(idx)
		} else if mod&fyne.KeyModifierShift != 0 {
			p.selectRange(idx)
		} else {
			p.selectTrack(idx)
		}
	} else {
		p.selectTrack(idx)
	}
	p.Refresh()
}

func (p *PlayQueueList) selectTrack(idx int) {
	p.tracksMutex.RLock()
	defer p.tracksMutex.RUnlock()
	util.SelectItem(p.items, idx)
}

func (p *PlayQueueList) selectAddOrRemove(idx int) {
	p.tracksMutex.RLock()
	defer p.tracksMutex.RUnlock()
	p.items[idx].Selected = !p.items[idx].Selected
}

func (p *PlayQueueList) selectRange(idx int) {
	p.tracksMutex.RLock()
	defer p.tracksMutex.RUnlock()
	util.SelectItemRange(p.items, idx)
}

func (p *PlayQueueList) onShowContextMenu(e *fyne.PointEvent, trackIdx int) {
	p.selectTrack(trackIdx)
	p.list.Refresh()
	selected := p.selectedItems()

	allTracks := true
	for _, item := range selected {
		if item.Metadata().Type == mediaprovider.MediaItemTypeRadioStation {
			allTracks = false
			break
		}
	}

	var menu *widget.PopUpMenu
	if allTracks {
		p.ensureTracksMenu()
		p.ratingSubmenu.Disabled = p.DisableRating
		p.infoMenuItem.Disabled = len(selected) != 1
		p.shareMenuItem.Disabled = p.DisableSharing || len(selected) != 1
		menu = p.menu
	} else {
		p.ensureRadiosMenu()
		menu = p.radiosMenu
	}
	menu.ShowAtPosition(e.AbsolutePosition)
}

func (p *PlayQueueList) ensureTracksMenu() {
	if p.menu != nil {
		return
	}
	var menuItems []*fyne.MenuItem
	if p.useNonQueueMenu {
		menuItems = append(menuItems, p.createQueueActionMenuItems()...)
		menuItems = append(menuItems, fyne.NewMenuItemSeparator())
	}

	playlist := fyne.NewMenuItem(lang.L("Add to playlist")+"...", func() {
		if p.OnAddToPlaylist != nil {
			p.OnAddToPlaylist(p.selectedItemIDs())
		}
	})
	playlist.Icon = myTheme.PlaylistIcon
	download := fyne.NewMenuItem(lang.L("Download")+"...", func() {
		if p.OnDownload != nil {
			p.OnDownload(p.selectedTracks(), "Selected tracks")
		}
	})
	download.Icon = theme.DownloadIcon()
	p.infoMenuItem = fyne.NewMenuItem(lang.L("Show info")+"...", func() {
		if p.OnShowTrackInfo != nil {
			p.OnShowTrackInfo(p.selectedTracks()[0])
		}
	})
	p.infoMenuItem.Icon = theme.InfoIcon()
	p.shareMenuItem = fyne.NewMenuItem(lang.L("Share")+"...", func() {
		if p.OnShare != nil {
			p.OnShare(p.selectedTracks())
		}
	})
	p.shareMenuItem.Icon = myTheme.ShareIcon
	favorite := fyne.NewMenuItem(lang.L("Set favorite"), func() {
		if p.OnSetFavorite != nil {
			p.OnSetFavorite(p.selectedItemIDs(), true)
		}
	})
	favorite.Icon = myTheme.FavoriteIcon
	unfavorite := fyne.NewMenuItem(lang.L("Unset favorite"), func() {
		if p.OnSetFavorite != nil {
			p.OnSetFavorite(p.selectedItemIDs(), false)
		}
	})
	unfavorite.Icon = myTheme.NotFavoriteIcon
	p.ratingSubmenu = util.NewRatingSubmenu(func(rating int) {
		if p.OnSetRating != nil {
			p.OnSetRating(p.selectedItemIDs(), rating)
		}
	})
	menuItems = append(menuItems,
		playlist,
		download,
		p.infoMenuItem,
		p.shareMenuItem,
		fyne.NewMenuItemSeparator(),
		favorite,
		unfavorite,
		p.ratingSubmenu,
	)

	if !p.useNonQueueMenu {
		remove := fyne.NewMenuItem(lang.L("Remove from queue"), func() {
			if p.OnRemoveFromQueue != nil {
				p.OnRemoveFromQueue(p.selectedIdxs())
			}
		})
		remove.Icon = theme.ContentRemoveIcon()
		menuItems = append(menuItems,
			fyne.NewMenuItemSeparator(),
			remove)
	}

	p.menu = widget.NewPopUpMenu(
		fyne.NewMenu("", menuItems...),
		fyne.CurrentApp().Driver().CanvasForObject(p),
	)
}

func (p *PlayQueueList) ensureRadiosMenu() {
	if p.radiosMenu != nil {
		return
	}
	remove := fyne.NewMenuItem(lang.L("Remove from queue"), func() {
		if p.OnRemoveFromQueue != nil {
			p.OnRemoveFromQueue(p.selectedIdxs())
		}
	})
	remove.Icon = theme.ContentRemoveIcon()
	p.radiosMenu = widget.NewPopUpMenu(
		fyne.NewMenu("", remove),
		fyne.CurrentApp().Driver().CanvasForObject(p),
	)
}

func (p *PlayQueueList) createQueueActionMenuItems() []*fyne.MenuItem {
	play := fyne.NewMenuItem(lang.L("Play"), func() {
		if p.OnPlaySelection != nil {
			p.OnPlaySelection(p.selectedItems(), false)
		}
	})
	play.Icon = theme.MediaPlayIcon()
	shuffle := fyne.NewMenuItem(lang.L("Shuffle"), func() {
		if p.OnPlaySelection != nil {
			p.OnPlaySelection(p.selectedItems(), true)
		}
	})
	shuffle.Icon = myTheme.ShuffleIcon
	playNext := fyne.NewMenuItem(lang.L("Play next"), func() {
		if p.OnPlaySelection != nil {
			p.OnPlaySelectionNext(p.selectedItems())
		}
	})
	playNext.Icon = myTheme.PlayNextIcon
	add := fyne.NewMenuItem(lang.L("Add to queue"), func() {
		if p.OnPlaySelection != nil {
			p.OnAddToQueue(p.selectedItems())
		}
	})
	add.Icon = theme.ContentAddIcon()
	return []*fyne.MenuItem{play, shuffle, playNext, add}
}

func (t *PlayQueueList) selectedItems() []mediaprovider.MediaItem {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedItems(t.items)
}

func (t *PlayQueueList) selectedTracks() []*mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedTracks(t.items)
}

func (t *PlayQueueList) selectedItemIDs() []string {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedItemIDs(t.items)
}

func (t *PlayQueueList) selectedIdxs() []int {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedIndexes(t.items)
}

func (p *PlayQueueList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.list)
}

type PlayQueueListRow struct {
	FocusListRowBase

	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	imageLoader   util.ThumbnailLoader
	playQueueList *PlayQueueList
	trackID       string
	isPlaying     bool

	playingIcon fyne.CanvasObject
	num         *widget.Label
	cover       *ImagePlaceholder
	title       *widget.Label
	artist      *MultiHyperlink
	time        *widget.Label
}

func NewPlayQueueListRow(playQueueList *PlayQueueList, im *backend.ImageManager, playingIcon fyne.CanvasObject) *PlayQueueListRow {
	p := &PlayQueueListRow{
		playingIcon:   playingIcon,
		playQueueList: playQueueList,
		num:           widget.NewLabel(""),
		cover:         NewImagePlaceholder(myTheme.TracksIcon, playQueueListThumbnailSize),
		title:         util.NewTruncatingLabel(),
		artist:        NewMultiHyperlink(),
		time:          util.NewTrailingAlignLabel(),
	}
	p.ExtendBaseWidget(p)

	p.cover.ScaleMode = canvas.ImageScaleFastest
	p.artist.OnTapped = playQueueList.onArtistTapped
	p.OnDoubleTapped = func() {
		playQueueList.onPlayTrackAt(p.ItemID())
	}
	p.OnTapped = func() {
		playQueueList.onSelectTrack(p.ItemID())
	}
	p.OnTappedSecondary = playQueueList.onShowContextMenu
	p.OnFocusNeighbor = func(up bool) {
		playQueueList.list.FocusNeighbor(p.ItemID(), up)
	}

	p.imageLoader = util.NewThumbnailLoader(im, func(i image.Image) {
		p.cover.SetImage(i, false)
	})
	p.imageLoader.OnBeforeLoad = func() {
		p.cover.SetImage(nil, false)
	}

	p.Content = container.New(playQueueList.colLayout,
		container.NewCenter(p.num),
		container.NewPadded(p.cover),
		container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-15),
			p.title, p.artist),
		container.NewCenter(p.time),
	)
	return p
}

func (p *PlayQueueListRow) TappedSecondary(e *fyne.PointEvent) {
	if p.OnTappedSecondary != nil {
		p.OnTappedSecondary(e, p.ListItemID)
	}
}

func (p *PlayQueueListRow) Update(tm *util.TrackListModel, rowNum int) {
	changed := false
	if tm.Selected != p.Selected {
		p.Selected = tm.Selected
		changed = true
	}

	if num := strconv.Itoa(rowNum); p.num.Text != num {
		p.num.Text = num
		changed = true
	}

	// Update info that can change if this row is bound to
	// a new track (*mediaprovider.Track)
	meta := tm.Item.Metadata()
	if meta.ID != p.trackID {
		if meta.Type == mediaprovider.MediaItemTypeRadioStation {
			p.cover.PlaceholderIcon = myTheme.RadioIcon
		} else {
			p.cover.PlaceholderIcon = myTheme.TracksIcon
		}
		p.imageLoader.Load(meta.CoverArtID)
		p.EnsureUnfocused()
		p.trackID = meta.ID
		p.title.Text = meta.Name
		p.artist.BuildSegments(meta.Artists, meta.ArtistIDs)
		p.time.Text = util.SecondsToMMSS(float64(meta.Duration))
		changed = true
	}

	// Render whether track is playing or not
	if isPlaying := p.playQueueList.nowPlayingID == meta.ID; isPlaying != p.isPlaying {
		p.isPlaying = isPlaying
		p.title.TextStyle.Bold = isPlaying

		if isPlaying {
			p.Content.(*fyne.Container).Objects[0] = p.playingIcon
		} else {
			p.Content.(*fyne.Container).Objects[0] = container.NewCenter(p.num)
		}
		changed = true
	}

	if changed {
		p.Refresh()
	}
}
