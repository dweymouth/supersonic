package widgets

import (
	"log"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/os"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TracklistSort struct {
	SortOrder  SortType
	ColumnName string
}

type TracklistOptions struct {
	// AutoNumber sets whether to auto-number the tracks 1..N in display order,
	// or to use the number from the track's metadata
	AutoNumber bool

	// Reorderable sets whether the tracklist supports drag-and-drop reordering.
	Reorderable bool

	// ShowDiscNumber sets whether to display the disc number as part of the '#' column,
	// (with format %d.%02d). Only applies if AutoNumber==false.
	ShowDiscNumber bool

	// AuxiliaryMenuItems sets additional menu items appended to the context menu
	// must be set before the context menu is shown for the first time
	AuxiliaryMenuItems []*fyne.MenuItem

	// DisablePlaybackMenu sets whether to disable playback options in
	// the tracklist context menu.
	DisablePlaybackMenu bool

	// Disables sorting the tracklist by clicking individual columns.
	DisableSorting bool

	// Disables the five star rating widget.
	DisableRating bool

	// Disables the sharing option.
	DisableSharing bool
}

type Tracklist struct {
	widget.BaseWidget

	compactRows bool
	columns     []TracklistColumn

	Options TracklistOptions

	// user action callbacks
	OnPlayTrackAt       func(int)
	OnPlaySelection     func(tracks []*mediaprovider.Track, shuffle bool)
	OnPlaySelectionNext func(trackIDs []*mediaprovider.Track)
	OnAddToQueue        func(trackIDs []*mediaprovider.Track)
	OnAddToPlaylist     func(trackIDs []string)
	OnSetFavorite       func(trackIDs []string, fav bool)
	OnSetRating         func(trackIDs []string, rating int)
	OnDownload          func(tracks []*mediaprovider.Track, downloadName string)
	OnShare             func(trackID string)
	OnPlaySongRadio     func(track *mediaprovider.Track)
	OnReorderTracks     func(trackIDs []string, insertPos int)
	OnShowTrackInfo     func(track *mediaprovider.Track)

	OnShowArtistPage func(artistID string)
	OnShowAlbumPage  func(albumID string)

	OnColumnVisibilityMenuShown func(*widget.PopUp)
	OnVisibleColumnsChanged     func([]string)
	OnTrackShown                func(tracknum int)

	visibleColumns []bool
	sorting        TracklistSort

	tracksMutex     sync.RWMutex
	tracks          []*util.TrackListModel
	tracksOrigOrder []*util.TrackListModel

	nowPlayingID      string
	colLayout         *layouts.ColumnsLayout
	hdr               *ListHeader
	list              *FocusList
	ctxMenu           *fyne.Menu
	ratingSubmenu     *fyne.MenuItem
	shareMenuItem     *fyne.MenuItem
	songRadioMenuItem *fyne.MenuItem
	infoMenuItem      *fyne.MenuItem
	container         *fyne.Container
}

func NewTracklist(tracks []*mediaprovider.Track, im *backend.ImageManager, useCompactRows bool) *Tracklist {
	initTracklistColumns()
	playIcon := theme.NewThemedResource(theme.MediaPlayIcon())
	playIcon.ColorName = theme.ColorNamePrimary

	t := &Tracklist{compactRows: useCompactRows}
	t.ExtendBaseWidget(t)
	t.columns = ExpandedTracklistRowColumns
	colWidths := ExpandedTracklistRowColumnWidths
	var playingIcon fyne.CanvasObject
	if useCompactRows {
		t.columns = CompactTracklistRowColumns
		colWidths = CompactTracklistRowColumnWidths
		playingIcon = container.NewCenter(container.NewHBox(util.NewHSpace(2), widget.NewIcon(playIcon)))
	} else {
		playIconImg := canvas.NewImageFromResource(playIcon)
		playIconImg.FillMode = canvas.ImageFillContain
		playIconImg.SetMinSize(fyne.NewSquareSize(theme.IconInlineSize() * 1.5))
		playingIcon = container.NewCenter(playIconImg)
	}
	t.visibleColumns = make([]bool, len(t.columns))

	if len(tracks) > 0 {
		t._setTracks(tracks)
	}

	t.colLayout = layouts.NewColumnsLayout(colWidths)
	t.hdr = NewListHeader(sharedutil.MapSlice(t.columns,
		func(t TracklistColumn) ListColumn { return t.Col }), t.colLayout)
	t.hdr.OnColumnSortChanged = t.onSorted
	t.hdr.OnColumnVisibilityChanged = t.setColumnVisible
	t.hdr.OnColumnVisibilityMenuShown = func(pop *widget.PopUp) {
		if t.OnColumnVisibilityMenuShown != nil {
			t.OnColumnVisibilityMenuShown(pop)
		}
	}

	t.list = NewFocusList(
		t.lenTracks,
		func() fyne.CanvasObject {
			var tr TracklistRow
			if t.compactRows {
				tr = NewCompactTracklistRow(t, playingIcon)
			} else {
				tr = NewExpandedTracklistRow(t, im, playingIcon)
			}
			tr.SetOnTapped(func() {
				t.onSelectTrack(tr.ItemID())
			})
			tr.SetOnTappedSecondary(t.onShowContextMenu)
			tr.SetOnDoubleTapped(func() {
				t.onPlayTrackAt(tr.ItemID())
			})
			tr.SetOnFocusNeighbor(func(up bool) {
				t.list.FocusNeighbor(tr.ItemID(), up)
			})
			return tr
		},
		func(itemID widget.ListItemID, item fyne.CanvasObject) {
			t.tracksMutex.RLock()
			// we could have removed tracks from the list in between
			// Fyne calling the length callback and this update callback
			// so the itemID may be out of bounds. if so, do nothing.
			if itemID >= len(t.tracks) {
				t.tracksMutex.RUnlock()
				return
			}
			model := t.tracks[itemID]
			t.tracksMutex.RUnlock()

			tr := item.(TracklistRow)
			if tr.TrackID() != model.Item.Metadata().ID || tr.ItemID() != itemID {
				tr.SetItemID(itemID)
			}
			i := -1 // signal that we want to display the actual track num.
			if t.Options.AutoNumber {
				i = itemID + 1
			}
			tr.Update(model, i)
			if t.OnTrackShown != nil {
				t.OnTrackShown(itemID)
			}
		})
	t.list.OnDragBegin = func(id int) {
		if !t.tracks[id].Selected {
			t.selectTrack(id)
			t.list.Refresh()
		}
	}
	t.list.OnDragEnd = func(dragged, insertPos int) {
		if t.OnReorderTracks != nil {
			t.OnReorderTracks(t.SelectedTrackIDs(), insertPos)
		}
	}
	t.container = container.NewBorder(t.hdr, nil, nil, nil, t.list)
	return t
}

func (t *Tracklist) Reset() {
	t.Clear()
	t.Options = TracklistOptions{}
	t.ctxMenu = nil
	t.SetSorting(TracklistSort{})
}

func (t *Tracklist) Scroll(amount float32) {
	t.list.ScrollToOffset(t.list.GetScrollOffset() + amount)
}

// Gets the track at the given index. Thread-safe.
func (t *Tracklist) TrackAt(idx int) *mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if idx >= len(t.tracks) {
		log.Println("error: Tracklist.TrackAt: index out of range")
		return nil
	}
	return t.tracks[idx].Track()
}

func (t *Tracklist) SetVisibleColumns(cols []string) {
	t.visibleColumns[0] = true
	t.visibleColumns[1] = true

	for i := 2; i < len(t.columns); i++ {
		t.visibleColumns[i] = false
		t.hdr.SetColumnVisible(i, false)
	}
	for _, col := range cols {
		if num := t.ColNumber(col); num < 0 {
			log.Printf("Unknown tracklist column %q", col)
		} else {
			t.visibleColumns[num] = true
			t.hdr.SetColumnVisible(num, true)
		}
	}
}

func (t *Tracklist) VisibleColumns() []string {
	var cols []string
	for i := 2; i < len(t.visibleColumns); i++ {
		if t.visibleColumns[i] {
			cols = append(cols, string(t.colName(i)))
		}
	}
	return cols
}

func (t *Tracklist) setColumnVisible(colNum int, vis bool) {
	if colNum >= len(t.visibleColumns) {
		log.Printf("error: Tracklist.SetColumnVisible: column index %d out of range", colNum)
		return
	}
	t.visibleColumns[colNum] = vis
	t.list.Refresh()
	if t.OnVisibleColumnsChanged != nil {
		t.OnVisibleColumnsChanged(t.VisibleColumns())
	}
}

func (t *Tracklist) Sorting() TracklistSort {
	return t.sorting
}

func (t *Tracklist) SetSorting(sorting TracklistSort) {
	if sorting.ColumnName == "" {
		// nil case - reset current sort
		if slices.ContainsFunc(t.columns, func(c TracklistColumn) bool {
			return c.Name == t.sorting.ColumnName
		}) {
			t.hdr.SetSorting(ListHeaderSort{ColNumber: t.ColNumber(t.sorting.ColumnName), Type: SortNone})
		}
		return
	}
	// actual sorting will be handled in callback from header
	t.hdr.SetSorting(ListHeaderSort{ColNumber: t.ColNumber(sorting.ColumnName), Type: sorting.SortOrder})
}

// Sets the currently playing track ID and updates the list rendering
func (t *Tracklist) SetNowPlaying(trackID string) {
	prevNowPlaying := t.nowPlayingID
	t.tracksMutex.RLock()
	trPrev, idxPrev := util.FindItemByID(t.tracks, prevNowPlaying)
	tr, idx := util.FindItemByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	t.nowPlayingID = trackID
	if trPrev != nil {
		t.list.RefreshItem(idxPrev)
	}
	if tr != nil {
		t.list.RefreshItem(idx)
	}
}

// Increments the play count of the given track and updates the list rendering
func (t *Tracklist) IncrementPlayCount(trackID string) {
	t.tracksMutex.RLock()
	tr, idx := util.FindItemByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	if tr != nil {
		tr.(*mediaprovider.Track).PlayCount += 1
		t.list.RefreshItem(idx)
	}
}

// Remove all tracks from the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) Clear() {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracks = nil
	t.tracksOrigOrder = nil
}

// Sets the tracks in the tracklist. Thread-safe.
func (t *Tracklist) SetTracks(trs []*mediaprovider.Track) {
	t._setTracks(trs)
	t.Refresh()
}

func (t *Tracklist) _setTracks(trs []*mediaprovider.Track) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracksOrigOrder = util.ToTrackListModels(trs)
	t.doSortTracks()
}

// Returns the tracks in the tracklist in the current display order.
func (t *Tracklist) GetTracks() []*mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return sharedutil.MapSlice(t.tracks, func(tm *util.TrackListModel) *mediaprovider.Track {
		return tm.Track()
	})
}

// Append more tracks to the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) AppendTracks(trs []*mediaprovider.Track) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracksOrigOrder = append(t.tracks, util.ToTrackListModels(trs)...)
	t.doSortTracks()
}

func (t *Tracklist) SelectAll() {
	t.tracksMutex.RLock()
	util.SelectAllItems(t.tracks)
	t.tracksMutex.RUnlock()
	t.list.Refresh()
}

func (t *Tracklist) UnselectAll() {
	t.unselectAll()
	t.list.Refresh()
}

func (t *Tracklist) unselectAll() {
	t.tracksMutex.RLock()
	util.UnselectAllItems(t.tracks)
	t.tracksMutex.RUnlock()
}

func (t *Tracklist) SelectAndScrollToTrack(trackID string) {
	t.tracksMutex.RLock()
	idx := -1
	for i, tr := range t.tracks {
		if tr.Item.Metadata().ID == trackID {
			idx = i
			tr.Selected = true
		} else {
			tr.Selected = false
		}
	}
	t.tracksMutex.RUnlock()
	if idx >= 0 {
		t.list.ScrollTo(idx)
	}
}

func (t *Tracklist) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}

func (t *Tracklist) Refresh() {
	t.list.EnableDragging = t.Options.Reorderable
	t.hdr.DisableSorting = t.Options.DisableSorting
	t.BaseWidget.Refresh()
}

// do nothing Tapped handler so that tapping the separator between rows
// doesn't fall through to the page (which calls UnselectAll on tracklist)
func (t *Tracklist) Tapped(*fyne.PointEvent) {}

func (t *Tracklist) stringSort(fieldFn func(*util.TrackListModel) string) {
	new := make([]*util.TrackListModel, len(t.tracksOrigOrder))
	copy(new, t.tracksOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		cmp := strings.Compare(fieldFn(new[i]), fieldFn(new[j]))
		if t.sorting.SortOrder == SortDescending {
			return cmp > 0
		}
		return cmp < 0
	})
	t.tracks = new
}

func (t *Tracklist) intSort(fieldFn func(*util.TrackListModel) int64) {
	new := make([]*util.TrackListModel, len(t.tracksOrigOrder))
	copy(new, t.tracksOrigOrder)
	sort.SliceStable(new, func(i, j int) bool {
		if t.sorting.SortOrder == SortDescending {
			return fieldFn(new[i]) > fieldFn(new[j])
		}
		return fieldFn(new[i]) < fieldFn(new[j])
	})
	t.tracks = new
}

func (t *Tracklist) doSortTracks() {
	if t.sorting.SortOrder == SortNone {
		t.tracks = t.tracksOrigOrder
		return
	}
	switch t.sorting.ColumnName {
	case ColumnNum:
		if t.sorting.SortOrder == SortDescending {
			t.tracks = sharedutil.Reversed(t.tracksOrigOrder)
		} else {
			t.tracks = t.tracksOrigOrder
		}
	case ColumnTitle, ColumnTitleArtist:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track().Title })
	case ColumnArtist:
		t.stringSort(func(tr *util.TrackListModel) string { return strings.Join(tr.Track().ArtistNames, ", ") })
	case ColumnAlbum:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track().Album })
	case ColumnPath:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track().FilePath })
	case ColumnRating:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track().Rating) })
	case ColumnTime:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track().Duration) })
	case ColumnYear:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track().Year) })
	case ColumnSize:
		t.intSort(func(tr *util.TrackListModel) int64 { return tr.Track().Size })
	case ColumnPlays:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track().PlayCount) })
	case ColumnComment:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track().Comment })
	case ColumnBitrate:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track().BitRate) })
	case ColumnFavorite:
		t.intSort(func(tr *util.TrackListModel) int64 {
			if tr.Track().Favorite {
				return 1
			}
			return 0
		})
	}
}

func (t *Tracklist) onSorted(sort ListHeaderSort) {
	t.sorting = TracklistSort{ColumnName: t.colName(sort.ColNumber), SortOrder: sort.Type}
	t.tracksMutex.Lock()
	t.doSortTracks()
	t.tracksMutex.Unlock()
	t.Refresh()
}

func (t *Tracklist) onPlayTrackAt(idx int) {
	if t.OnPlayTrackAt != nil {
		t.OnPlayTrackAt(idx)
	}
}

func (t *Tracklist) onSelectTrack(idx int) {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		mod := d.CurrentKeyModifiers()
		if mod&os.ControlModifier != 0 {
			t.selectAddOrRemove(idx)
		} else if mod&fyne.KeyModifierShift != 0 {
			t.selectRange(idx)
		} else {
			t.selectTrack(idx)
		}
	} else {
		t.selectTrack(idx)
	}
	t.list.Refresh()
}

func (t *Tracklist) selectAddOrRemove(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	t.tracks[idx].Selected = !t.tracks[idx].Selected
}

func (t *Tracklist) selectTrack(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	util.SelectItem(t.tracks, idx)
}

func (t *Tracklist) selectRange(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	util.SelectItemRange(t.tracks, idx)
}

func (t *Tracklist) onShowContextMenu(e *fyne.PointEvent, trackIdx int) {
	t.selectTrack(trackIdx)
	t.list.Refresh()
	if t.ctxMenu == nil {
		t.ctxMenu = fyne.NewMenu("")
		if !t.Options.DisablePlaybackMenu {
			play := fyne.NewMenuItem("Play", func() {
				if t.OnPlaySelection != nil {
					t.OnPlaySelection(t.selectedTracks(), false)
				}
			})
			play.Icon = theme.MediaPlayIcon()
			shuffle := fyne.NewMenuItem("Shuffle", func() {
				if t.OnPlaySelection != nil {
					t.OnPlaySelection(t.selectedTracks(), true)
				}
			})
			shuffle.Icon = myTheme.ShuffleIcon
			playNext := fyne.NewMenuItem("Play next", func() {
				if t.OnPlaySelection != nil {
					t.OnPlaySelectionNext(t.selectedTracks())
				}
			})
			playNext.Icon = myTheme.PlayNextIcon
			add := fyne.NewMenuItem("Add to queue", func() {
				if t.OnPlaySelection != nil {
					t.OnAddToQueue(t.selectedTracks())
				}
			})
			add.Icon = theme.ContentAddIcon()
			t.songRadioMenuItem = fyne.NewMenuItem("Play song radio", func() {
				t.onPlaySongRadio(t.selectedTracks())
			})
			t.songRadioMenuItem.Icon = myTheme.RadioIcon
			t.ctxMenu.Items = append(t.ctxMenu.Items,
				play, shuffle, playNext, add, t.songRadioMenuItem)
		}
		playlist := fyne.NewMenuItem("Add to playlist...", func() {
			if t.OnAddToPlaylist != nil {
				t.OnAddToPlaylist(t.SelectedTrackIDs())
			}
		})
		playlist.Icon = myTheme.PlaylistIcon
		download := fyne.NewMenuItem("Download...", func() {
			t.onDownload(t.selectedTracks(), "Selected tracks")
		})
		download.Icon = theme.DownloadIcon()
		t.infoMenuItem = fyne.NewMenuItem("Show info...", func() {
			if t.OnShowTrackInfo != nil {
				t.OnShowTrackInfo(t.selectedTracks()[0])
			}
		})
		t.infoMenuItem.Icon = theme.InfoIcon()
		favorite := fyne.NewMenuItem("Set favorite", func() {
			t.onSetFavorites(t.selectedTracks(), true, true)
		})
		favorite.Icon = myTheme.FavoriteIcon
		unfavorite := fyne.NewMenuItem("Unset favorite", func() {
			t.onSetFavorites(t.selectedTracks(), false, true)
		})
		unfavorite.Icon = myTheme.NotFavoriteIcon
		t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator(),
			playlist, download, t.infoMenuItem)
		t.shareMenuItem = fyne.NewMenuItem("Share...", func() {
			t.onShare(t.selectedTracks())
		})
		t.shareMenuItem.Icon = myTheme.ShareIcon
		t.ctxMenu.Items = append(t.ctxMenu.Items, t.shareMenuItem)
		t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
		t.ctxMenu.Items = append(t.ctxMenu.Items, favorite, unfavorite)
		t.ratingSubmenu = util.NewRatingSubmenu(func(rating int) {
			t.onSetRatings(t.selectedTracks(), rating, true)
		})
		t.ctxMenu.Items = append(t.ctxMenu.Items, t.ratingSubmenu)
		if len(t.Options.AuxiliaryMenuItems) > 0 {
			t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
			t.ctxMenu.Items = append(t.ctxMenu.Items, t.Options.AuxiliaryMenuItems...)
		}
	}
	t.ratingSubmenu.Disabled = t.Options.DisableRating
	t.shareMenuItem.Disabled = t.Options.DisableSharing || len(t.selectedTracks()) != 1
	t.infoMenuItem.Disabled = len(t.selectedTracks()) != 1
	widget.ShowPopUpMenuAtPosition(t.ctxMenu, fyne.CurrentApp().Driver().CanvasForObject(t), e.AbsolutePosition)
}

func (t *Tracklist) onSetFavorite(trackID string, fav bool) {
	t.tracksMutex.RLock()
	item, _ := util.FindItemByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	t.onSetFavorites([]*mediaprovider.Track{item.(*mediaprovider.Track)}, fav, false)
}

func (t *Tracklist) onSetFavorites(tracks []*mediaprovider.Track, fav bool, needRefresh bool) {
	for _, tr := range tracks {
		tr.Favorite = fav
	}
	if needRefresh {
		t.Refresh()
	}
	// notify listener
	if t.OnSetFavorite != nil {
		t.OnSetFavorite(sharedutil.TracksToIDs(tracks), fav)
	}
}

func (t *Tracklist) onSetRating(trackID string, rating int) {
	// update our own track model
	t.tracksMutex.RLock()
	item, _ := util.FindItemByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	t.onSetRatings([]*mediaprovider.Track{item.(*mediaprovider.Track)}, rating, false)
}

func (t *Tracklist) onSetRatings(tracks []*mediaprovider.Track, rating int, needRefresh bool) {
	for _, tr := range tracks {
		tr.Rating = rating
	}
	if needRefresh {
		t.Refresh()
	}
	// notify listener
	if t.OnSetRating != nil {
		t.OnSetRating(sharedutil.TracksToIDs(tracks), rating)
	}
}

func (t *Tracklist) onArtistTapped(artistID string) {
	if t.OnShowArtistPage != nil {
		t.OnShowArtistPage(artistID)
	}
}

func (t *Tracklist) onAlbumTapped(albumID string) {
	if t.OnShowAlbumPage != nil {
		t.OnShowAlbumPage(albumID)
	}
}

func (t *Tracklist) onDownload(tracks []*mediaprovider.Track, downloadName string) {
	if t.OnDownload != nil {
		t.OnDownload(tracks, downloadName)
	}
}

func (t *Tracklist) onShare(tracks []*mediaprovider.Track) {
	if t.OnShare != nil {
		if len(tracks) > 0 {
			t.OnShare(tracks[0].ID)
		}
	}
}

func (t *Tracklist) onPlaySongRadio(tracks []*mediaprovider.Track) {
	if t.OnPlaySongRadio != nil {
		if len(tracks) > 0 {
			t.OnPlaySongRadio(tracks[0])
		}
	}
}

func (t *Tracklist) selectedTracks() []*mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedTracks(t.tracks)
}

func (t *Tracklist) SelectedTrackIDs() []string {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return util.SelectedItemIDs(t.tracks)
}

func (t *Tracklist) lenTracks() int {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return len(t.tracks)
}

func (t *Tracklist) ColNumber(colName string) int {
	i := slices.IndexFunc(t.columns, func(c TracklistColumn) bool {
		return c.Name == colName
	})
	if i < 0 {
		log.Printf("error: Tracklist: invalid column name %s", colName)
	}
	return i
}

func (t *Tracklist) colName(i int) string {
	if i < len(t.columns) {
		return t.columns[i].Name
	}
	log.Println("notReached: Tracklist.colName")
	return ""
}
