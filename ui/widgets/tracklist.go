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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	ColumnNum      = "Num"
	ColumnTitle    = "Title"
	ColumnArtist   = "Artist"
	ColumnAlbum    = "Album"
	ColumnTime     = "Time"
	ColumnYear     = "Year"
	ColumnFavorite = "Favorite"
	ColumnRating   = "Rating"
	ColumnPlays    = "Plays"
	ColumnComment  = "Comment"
	ColumnBitrate  = "Bitrate"
	ColumnSize     = "Size"
	ColumnPath     = "Path"

	numColumns = 13
)

var columns = []string{
	ColumnNum, ColumnTitle, ColumnArtist, ColumnAlbum, ColumnTime, ColumnYear, ColumnFavorite,
	ColumnRating, ColumnPlays, ColumnComment, ColumnBitrate, ColumnSize, ColumnPath,
}

type TracklistSort struct {
	SortOrder  SortType
	ColumnName string
}

type TracklistOptions struct {
	// AutoNumber sets whether to auto-number the tracks 1..N in display order,
	// or to use the number from the track's metadata
	AutoNumber bool

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
	container         *fyne.Container
}

func NewTracklist(tracks []*mediaprovider.Track, im *backend.ImageManager, useCompactRows bool) *Tracklist {
	t := &Tracklist{visibleColumns: make([]bool, numColumns), compactRows: useCompactRows}
	t.ExtendBaseWidget(t)

	if len(tracks) > 0 {
		t._setTracks(tracks)
	}

	// #, Title/Artist, Album, Time, Year, Favorite, Rating, Plays, Comment, Bitrate, Size, Path
	colWidths := []float32{40, -1, -1, 60, 60, 55, 100, 65, -1, 75, 75, -1}
	if useCompactRows {
		// #, Title, Artist, Album, Time, Year, Favorite, Rating, Plays, Comment, Bitrate, Size, Path
		colWidths = []float32{40, -1, -1, -1, 60, 60, 55, 100, 65, -1, 75, 75, -1}
	}

	t.colLayout = layouts.NewColumnsLayout(colWidths)
	t.buildHeader()
	t.hdr.OnColumnSortChanged = t.onSorted
	t.hdr.OnColumnVisibilityChanged = t.setColumnVisible
	t.hdr.OnColumnVisibilityMenuShown = func(pop *widget.PopUp) {
		if t.OnColumnVisibilityMenuShown != nil {
			t.OnColumnVisibilityMenuShown(pop)
		}
	}

	playIcon := theme.NewThemedResource(theme.MediaPlayIcon())
	playIcon.ColorName = theme.ColorNamePrimary
	playingIcon := container.NewCenter(container.NewHBox(util.NewHSpace(2), widget.NewIcon(playIcon)))

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
			t.list.SetItemForID(itemID, tr)
			if tr.TrackID() != model.Track.ID || tr.ItemID() != itemID {
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

func (t *Tracklist) buildHeader() {
	cols := []ListColumn{{Text: "#", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false}}
	if t.compactRows {
		cols = append(cols,
			ListColumn{Text: "Title", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
			ListColumn{Text: "Artist", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		)
	} else {
		cols = append(cols,
			ListColumn{Text: "   Title / Aritst", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		)
	}
	cols = append(cols,
		ListColumn{Text: "Album", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		ListColumn{Text: "Time", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		ListColumn{Text: "Year", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		ListColumn{Text: " Fav.", Alignment: fyne.TextAlignCenter, CanToggleVisible: true},
		ListColumn{Text: "Rating", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		ListColumn{Text: "Plays", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		ListColumn{Text: "Comment", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		ListColumn{Text: "Bitrate", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		ListColumn{Text: "Size", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		ListColumn{Text: "File Path", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
	)

	t.hdr = NewListHeader(cols, t.colLayout)
}

// Gets the track at the given index. Thread-safe.
func (t *Tracklist) TrackAt(idx int) *mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if idx >= len(t.tracks) {
		log.Println("error: Tracklist.TrackAt: index out of range")
		return nil
	}
	return t.tracks[idx].Track
}

func (t *Tracklist) SetVisibleColumns(cols []string) {
	t.visibleColumns[0] = true
	t.visibleColumns[1] = true
	if !t.compactRows {
		// hard-code artist column to visible since it's part of title col
		t.visibleColumns[2] = true
	}

	l := len(t.visibleColumns)
	if !t.compactRows {
		l-- // expanded rows have one fewer column due to merged title/artist
	}
	for i := 2; i < l; i++ {
		t.hdr.SetColumnVisible(i, false)
	}
	for _, col := range cols {
		if num := ColNumber(col); num < 0 {
			log.Printf("Unknown tracklist column %q", col)
		} else {
			if num == 2 && !t.compactRows {
				// Artist column is hard-coded visible (part of title), skip
				continue
			}
			t.visibleColumns[num] = true
			if !t.compactRows && num >= 3 {
				num -= 1 // shift cols by 1 to account for Title/Artist combined col
			}
			t.hdr.SetColumnVisible(num, true)
		}
	}
}

func (t *Tracklist) VisibleColumns() []string {
	var cols []string
	for i := 2; i < len(t.visibleColumns); i++ {
		if t.visibleColumns[i] {
			cols = append(cols, string(t.colName(i, false)))
		}
	}
	return cols
}

func (t *Tracklist) setColumnVisible(colNum int, vis bool) {
	if colNum >= len(t.visibleColumns) {
		log.Printf("error: Tracklist.SetColumnVisible: column index %d out of range", colNum)
		return
	}
	if !t.compactRows && colNum >= 2 {
		colNum++ // account for off-by-one from merged title/artist column
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
		if slices.Contains(columns, t.sorting.ColumnName) {
			t.hdr.SetSorting(ListHeaderSort{ColNumber: ColNumber(t.sorting.ColumnName), Type: SortNone})
		}
		return
	}
	// actual sorting will be handled in callback from header
	t.hdr.SetSorting(ListHeaderSort{ColNumber: ColNumber(sorting.ColumnName), Type: sorting.SortOrder})
}

// Sets the currently playing track ID and updates the list rendering
func (t *Tracklist) SetNowPlaying(trackID string) {
	prevNowPlaying := t.nowPlayingID
	t.tracksMutex.RLock()
	trPrev, idxPrev := util.FindTrackByID(t.tracks, prevNowPlaying)
	tr, idx := util.FindTrackByID(t.tracks, trackID)
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
	tr, idx := util.FindTrackByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	if tr != nil {
		tr.PlayCount += 1
		t.list.RefreshItem(idx)
	}
}

// Remove all tracks from the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) Clear() {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracks = nil
	t.tracksOrigOrder = nil
	t.list.ClearItemForIDMap()
}

// Sets the tracks in the tracklist. Thread-safe.
func (t *Tracklist) SetTracks(trs []*mediaprovider.Track) {
	t._setTracks(trs)
	t.Refresh()
}

func (t *Tracklist) _setTracks(trs []*mediaprovider.Track) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	if t.list != nil {
		t.list.ClearItemForIDMap()
	}
	t.tracksOrigOrder = util.ToTrackListModels(trs)
	t.doSortTracks()
}

// Returns the tracks in the tracklist in the current display order.
func (t *Tracklist) GetTracks() []*mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return sharedutil.MapSlice(t.tracks, func(tm *util.TrackListModel) *mediaprovider.Track {
		return tm.Track
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
	util.SelectAllTracks(t.tracks)
	t.tracksMutex.RUnlock()
	t.list.Refresh()
}

func (t *Tracklist) UnselectAll() {
	t.unselectAll()
	t.list.Refresh()
}

func (t *Tracklist) unselectAll() {
	t.tracksMutex.RLock()
	util.UnselectAllTracks(t.tracks)
	t.tracksMutex.RUnlock()
}

func (t *Tracklist) SelectAndScrollToTrack(trackID string) {
	t.tracksMutex.RLock()
	idx := -1
	for i, tr := range t.tracks {
		if tr.Track.ID == trackID {
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
	case ColumnTitle:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track.Name })
	case ColumnArtist:
		t.stringSort(func(tr *util.TrackListModel) string { return strings.Join(tr.Track.ArtistNames, ", ") })
	case ColumnAlbum:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track.Album })
	case ColumnPath:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track.FilePath })
	case ColumnRating:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track.Rating) })
	case ColumnTime:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track.Duration) })
	case ColumnYear:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track.Year) })
	case ColumnSize:
		t.intSort(func(tr *util.TrackListModel) int64 { return tr.Track.Size })
	case ColumnPlays:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track.PlayCount) })
	case ColumnComment:
		t.stringSort(func(tr *util.TrackListModel) string { return tr.Track.Comment })
	case ColumnBitrate:
		t.intSort(func(tr *util.TrackListModel) int64 { return int64(tr.Track.BitRate) })
	case ColumnFavorite:
		t.intSort(func(tr *util.TrackListModel) int64 {
			if tr.Track.Favorite {
				return 1
			}
			return 0
		})
	}
}

func (t *Tracklist) onSorted(sort ListHeaderSort) {
	t.sorting = TracklistSort{ColumnName: t.colName(sort.ColNumber, true), SortOrder: sort.Type}
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
	util.SelectTrack(t.tracks, idx)
}

func (t *Tracklist) selectRange(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	util.SelectTrackRange(t.tracks, idx)
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
			t.songRadioMenuItem.Icon = myTheme.BroadcastIcon
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
		favorite := fyne.NewMenuItem("Set favorite", func() {
			t.onSetFavorites(t.selectedTracks(), true, true)
		})
		favorite.Icon = myTheme.FavoriteIcon
		unfavorite := fyne.NewMenuItem("Unset favorite", func() {
			t.onSetFavorites(t.selectedTracks(), false, true)
		})
		unfavorite.Icon = myTheme.NotFavoriteIcon
		t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator(),
			playlist, download)
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
	widget.ShowPopUpMenuAtPosition(t.ctxMenu, fyne.CurrentApp().Driver().CanvasForObject(t), e.AbsolutePosition)
}

func (t *Tracklist) onSetFavorite(trackID string, fav bool) {
	t.tracksMutex.RLock()
	tr, _ := util.FindTrackByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	t.onSetFavorites([]*mediaprovider.Track{tr}, fav, false)
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
	tr, _ := util.FindTrackByID(t.tracks, trackID)
	t.tracksMutex.RUnlock()
	t.onSetRatings([]*mediaprovider.Track{tr}, rating, false)
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
	return util.SelectedTrackIDs(t.tracks)
}

func (t *Tracklist) lenTracks() int {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return len(t.tracks)
}

func ColNumber(colName string) int {
	i := slices.Index(columns, colName)
	if i < 0 {
		log.Printf("error: Tracklist: invalid column name %s", colName)
	}
	return i
}

func (t *Tracklist) colName(i int, adjustForExpandedRows bool) string {
	if adjustForExpandedRows && !t.compactRows && i >= 2 {
		i += 1 // adjust for off-by-one from combined title/artist column
	}
	if i < len(columns) {
		return columns[i]
	}
	log.Println("notReached: Tracklist.colName")
	return ""
}
