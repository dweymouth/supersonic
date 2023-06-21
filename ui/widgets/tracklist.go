package widgets

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

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
	ColumnBitrate  = "Bitrate"
	ColumnSize     = "Size"
	ColumnPath     = "Path"
)

var columns = []string{
	ColumnNum, ColumnTitle, ColumnArtist, ColumnAlbum, ColumnTime, ColumnYear,
	ColumnFavorite, ColumnRating, ColumnPlays, ColumnBitrate, ColumnSize, ColumnPath,
}

type TracklistSort struct {
	SortOrder  SortType
	ColumnName string
}

type Tracklist struct {
	widget.BaseWidget

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

	// user action callbacks
	OnPlayTrackAt   func(int)
	OnPlaySelection func(tracks []*mediaprovider.Track, shuffle bool)
	OnAddToQueue    func(trackIDs []*mediaprovider.Track)
	OnAddToPlaylist func(trackIDs []string)
	OnSetFavorite   func(trackIDs []string, fav bool)
	OnSetRating     func(trackIDs []string, rating int)
	OnDownload      func(tracks []*mediaprovider.Track, downloadName string)

	OnShowArtistPage func(artistID string)
	OnShowAlbumPage  func(albumID string)

	OnColumnVisibilityMenuShown func(*widget.PopUp)
	OnVisibleColumnsChanged     func([]string)
	OnTrackShown                func(tracknum int)

	visibleColumns []bool
	sorting        TracklistSort

	tracksMutex     sync.RWMutex
	tracks          []*trackModel
	tracksOrigOrder []*trackModel

	nowPlayingID string
	colLayout    *layouts.ColumnsLayout
	hdr          *ListHeader
	list         *widget.List
	ctxMenu      *fyne.Menu
	container    *fyne.Container
}

type trackModel struct {
	track    *mediaprovider.Track
	selected bool
}

func NewTracklist(tracks []*mediaprovider.Track) *Tracklist {
	t := &Tracklist{visibleColumns: make([]bool, 12)}
	t.ExtendBaseWidget(t)

	if len(tracks) > 0 {
		t.SetTracks(tracks)
	}

	// #, Title, Artist, Album, Time, Year, Favorite, Rating, Plays, Bitrate, Size, Path
	t.colLayout = layouts.NewColumnsLayout([]float32{40, -1, -1, -1, 60, 60, 55, 100, 65, 75, 75, -1})
	t.buildHeader()
	t.hdr.OnColumnSortChanged = t.onSorted
	t.hdr.OnColumnVisibilityChanged = t.setColumnVisible
	t.hdr.OnColumnVisibilityMenuShown = func(pop *widget.PopUp) {
		if t.OnColumnVisibilityMenuShown != nil {
			t.OnColumnVisibilityMenuShown(pop)
		}
	}
	playingIcon := container.NewCenter(container.NewHBox(util.NewHSpace(2), widget.NewIcon(theme.MediaPlayIcon())))
	t.list = widget.NewList(
		t.lenTracks,
		func() fyne.CanvasObject {
			tr := NewTrackRow(t, playingIcon)
			tr.OnTapped = func() { t.onSelectTrack(tr.trackIdx) }
			tr.OnTappedSecondary = t.onShowContextMenu
			tr.OnDoubleTapped = func() { t.onPlayTrackAt(tr.trackIdx) }
			return tr
		},
		func(itemID widget.ListItemID, item fyne.CanvasObject) {
			tr := item.(*TrackRow)
			tr.trackIdx = itemID
			i := -1 // signal that we want to display the actual track num.
			if t.AutoNumber {
				i = itemID + 1
			}
			tr.Update(t.trackModelAt(itemID), i)
			if t.OnTrackShown != nil {
				t.OnTrackShown(itemID)
			}
		})
	t.container = container.NewBorder(t.hdr, nil, nil, nil, t.list)
	return t
}

func (t *Tracklist) buildHeader() {
	t.hdr = NewListHeader([]ListColumn{
		{Text: "#", Alignment: fyne.TextAlignTrailing, CanToggleVisible: false},
		{Text: "Title", Alignment: fyne.TextAlignLeading, CanToggleVisible: false},
		{Text: "Artist", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		{Text: "Album", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		{Text: "Time", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		{Text: "Year", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		{Text: " Fav.", Alignment: fyne.TextAlignCenter, CanToggleVisible: true},
		{Text: "Rating", Alignment: fyne.TextAlignLeading, CanToggleVisible: true},
		{Text: "Plays", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		{Text: "Bitrate", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		{Text: "Size", Alignment: fyne.TextAlignTrailing, CanToggleVisible: true},
		{Text: "File Path", Alignment: fyne.TextAlignLeading, CanToggleVisible: true}},
		t.colLayout)
}

// Gets the track at the given index. Thread-safe.
func (t *Tracklist) TrackAt(idx int) *mediaprovider.Track {
	return t.trackModelAt(idx).track
}

func (t *Tracklist) trackModelAt(idx int) *trackModel {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if idx >= len(t.tracks) {
		log.Println("error: Tracklist.trackModelAt: index out of range")
		return nil
	}
	return t.tracks[idx]
}

func (t *Tracklist) SetVisibleColumns(cols []string) {
	t.visibleColumns[0] = true
	t.visibleColumns[1] = true
	for i := 2; i < len(t.visibleColumns); i++ {
		t.visibleColumns[i] = false
		t.hdr.SetColumnVisible(i, false)
	}
	for _, col := range cols {
		if num := ColNumber(col); num < 0 {
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
			cols = append(cols, string(colName(i)))
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
		if sharedutil.SliceContains(columns, t.sorting.ColumnName) {
			t.hdr.SetSorting(ListHeaderSort{ColNumber: ColNumber(t.sorting.ColumnName), Type: SortNone})
		}
		return
	}
	// actual sorting will be handled in callback from header
	t.hdr.SetSorting(ListHeaderSort{ColNumber: ColNumber(sorting.ColumnName), Type: sorting.SortOrder})
}

func (t *Tracklist) SetNowPlaying(trackID string) {
	t.nowPlayingID = trackID
	t.list.Refresh()
}

func (t *Tracklist) IncrementPlayCount(trackID string) {
	t.tracksMutex.RLock()
	t.tracksMutex.RUnlock()
	if tr := t.findTrackByID(trackID); tr != nil {
		tr.PlayCount += 1
		t.list.Refresh()
	}
}

// Remove all tracks from the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) Clear() {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracks = nil
	t.tracksOrigOrder = nil
}

// Sets the tracks in the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) SetTracks(trs []*mediaprovider.Track) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracksOrigOrder = toTrackModels(trs)
	t.doSortTracks()
}

// Returns the tracks in the tracklist in the current display order.
func (t *Tracklist) GetTracks() []*mediaprovider.Track {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return sharedutil.MapSlice(t.tracks, func(tm *trackModel) *mediaprovider.Track {
		return tm.track
	})
}

// Append more tracks to the tracklist. Does not issue Refresh call. Thread-safe.
func (t *Tracklist) AppendTracks(trs []*mediaprovider.Track) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.tracksOrigOrder = append(t.tracks, toTrackModels(trs)...)
	t.doSortTracks()
}

func (t *Tracklist) SelectAll() {
	t.tracksMutex.RLock()
	for _, tm := range t.tracks {
		tm.selected = true
	}
	t.tracksMutex.RUnlock()
	t.list.Refresh()
}

func (t *Tracklist) UnselectAll() {
	t.unselectAll()
	t.list.Refresh()
}

func (t *Tracklist) unselectAll() {
	t.tracksMutex.RLock()
	for _, tm := range t.tracks {
		tm.selected = false
	}
	t.tracksMutex.RUnlock()
}

func (t *Tracklist) SelectAndScrollToTrack(trackID string) {
	t.tracksMutex.RLock()
	idx := -1
	for i, tr := range t.tracks {
		if tr.track.ID == trackID {
			idx = i
			tr.selected = true
		} else {
			tr.selected = false
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
	t.hdr.DisableSorting = t.DisableSorting
	t.BaseWidget.Refresh()
}

func toTrackModels(trs []*mediaprovider.Track) []*trackModel {
	return sharedutil.MapSlice(trs, func(tr *mediaprovider.Track) *trackModel {
		return &trackModel{track: tr, selected: false}
	})
}

// do nothing Tapped handler so that tapping the separator between rows
// doesn't fall through to the page (which calls UnselectAll on tracklist)
func (t *Tracklist) Tapped(*fyne.PointEvent) {}

func (t *Tracklist) stringSort(fieldFn func(*trackModel) string) {
	new := make([]*trackModel, len(t.tracksOrigOrder))
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

func (t *Tracklist) intSort(fieldFn func(*trackModel) int64) {
	new := make([]*trackModel, len(t.tracksOrigOrder))
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
		t.stringSort(func(tr *trackModel) string { return tr.track.Name })
	case ColumnArtist:
		t.stringSort(func(tr *trackModel) string { return tr.track.ArtistNames[0] })
	case ColumnAlbum:
		t.stringSort(func(tr *trackModel) string { return tr.track.Album })
	case ColumnPath:
		t.stringSort(func(tr *trackModel) string { return tr.track.FilePath })
	case ColumnRating:
		t.intSort(func(tr *trackModel) int64 { return int64(tr.track.Rating) })
	case ColumnTime:
		t.intSort(func(tr *trackModel) int64 { return int64(tr.track.Duration) })
	case ColumnYear:
		t.intSort(func(tr *trackModel) int64 { return int64(tr.track.Year) })
	case ColumnSize:
		t.intSort(func(tr *trackModel) int64 { return tr.track.Size })
	case ColumnPlays:
		t.intSort(func(tr *trackModel) int64 { return int64(tr.track.PlayCount) })
	case ColumnBitrate:
		t.intSort(func(tr *trackModel) int64 { return int64(tr.track.BitRate) })
	case ColumnFavorite:
		t.intSort(func(tr *trackModel) int64 {
			if tr.track.Favorite {
				return 1
			}
			return 0
		})
	}
}

func (t *Tracklist) onSorted(sort ListHeaderSort) {
	t.sorting = TracklistSort{ColumnName: colName(sort.ColNumber), SortOrder: sort.Type}
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
	t.tracks[idx].selected = !t.tracks[idx].selected
}

func (t *Tracklist) selectTrack(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if t.tracks[idx].selected {
		return
	}
	t.unselectAll()
	t.tracks[idx].selected = true
}

func (t *Tracklist) selectRange(idx int) {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if t.tracks[idx].selected {
		return
	}
	lastSelected := -1
	for i := len(t.tracks) - 1; i >= 0; i-- {
		if t.tracks[i].selected {
			lastSelected = i
			break
		}
	}
	if lastSelected < 0 {
		t.tracks[idx].selected = true
		return
	}
	from := minInt(idx, lastSelected)
	to := maxInt(idx, lastSelected)
	for i := from; i <= to; i++ {
		t.tracks[i].selected = true
	}
}

func (t *Tracklist) onShowContextMenu(e *fyne.PointEvent, trackIdx int) {
	t.selectTrack(trackIdx)
	t.list.Refresh()
	if t.ctxMenu == nil {
		t.ctxMenu = fyne.NewMenu("")
		if !t.DisablePlaybackMenu {
			t.ctxMenu.Items = append(t.ctxMenu.Items,
				fyne.NewMenuItem("Play", func() {
					if t.OnPlaySelection != nil {
						t.OnPlaySelection(t.selectedTracks(), false)
					}
				}))
			t.ctxMenu.Items = append(t.ctxMenu.Items,
				fyne.NewMenuItem("Shuffle", func() {
					if t.OnPlaySelection != nil {
						t.OnPlaySelection(t.selectedTracks(), true)
					}
				}))
			t.ctxMenu.Items = append(t.ctxMenu.Items,
				fyne.NewMenuItem("Add to queue", func() {
					if t.OnPlaySelection != nil {
						t.OnAddToQueue(t.selectedTracks())
					}
				}))
		}
		t.ctxMenu.Items = append(t.ctxMenu.Items,
			fyne.NewMenuItem("Add to playlist...", func() {
				if t.OnAddToPlaylist != nil {
					t.OnAddToPlaylist(t.SelectedTrackIDs())
				}
			}))
		t.ctxMenu.Items = append(t.ctxMenu.Items,
			fyne.NewMenuItem("Download", func() {
				t.onDownload(t.selectedTracks(), "Selected tracks")
			}))
		t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
		t.ctxMenu.Items = append(t.ctxMenu.Items,
			fyne.NewMenuItem("Set favorite", func() {
				t.onSetFavorites(t.selectedTracks(), true, true)
			}))
		t.ctxMenu.Items = append(t.ctxMenu.Items,
			fyne.NewMenuItem("Unset favorite", func() {
				t.onSetFavorites(t.selectedTracks(), false, true)
			}))
		ratingMenu := util.NewRatingSubmenu(func(rating int) {
			t.onSetRatings(t.selectedTracks(), rating, true)
		})
		t.ctxMenu.Items = append(t.ctxMenu.Items, ratingMenu)
		if len(t.AuxiliaryMenuItems) > 0 {
			t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
			t.ctxMenu.Items = append(t.ctxMenu.Items, t.AuxiliaryMenuItems...)
		}
	}
	widget.ShowPopUpMenuAtPosition(t.ctxMenu, fyne.CurrentApp().Driver().CanvasForObject(t), e.AbsolutePosition)
}

func (t *Tracklist) onSetFavorite(trackID string, fav bool) {
	t.tracksMutex.RLock()
	tr := t.findTrackByID(trackID)
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
	tr := t.findTrackByID(trackID)
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

func (t *Tracklist) findTrackByID(id string) *mediaprovider.Track {
	idx := sharedutil.Find(t.tracks, func(tr *trackModel) bool {
		return tr.track.ID == id
	})
	if idx >= 0 {
		return t.tracks[idx].track
	}
	return nil
}

func (t *Tracklist) selectedTrackModels() []*trackModel {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return sharedutil.FilterSlice(t.tracks, func(tm *trackModel) bool {
		return tm.selected
	})
}

func (t *Tracklist) selectedTracks() []*mediaprovider.Track {
	return sharedutil.MapSlice(t.selectedTrackModels(), func(tm *trackModel) *mediaprovider.Track {
		return tm.track
	})
}

func (t *Tracklist) SelectedTrackIDs() []string {
	return sharedutil.MapSlice(t.selectedTrackModels(), func(tm *trackModel) string {
		return tm.track.ID
	})
}

func (t *Tracklist) lenTracks() int {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return len(t.tracks)
}

func ColNumber(colName string) int {
	i := sharedutil.IndexOf(columns, colName)
	if i < 0 {
		log.Printf("error: Tracklist: invalid column name %s", colName)
	}
	return i
}

func colName(i int) string {
	if i < len(columns) {
		return columns[i]
	}
	log.Println("notReached: Tracklist.colName")
	return ""
}

type TrackRow struct {
	ListRowBase

	// internal state
	tracklist  *Tracklist
	trackIdx   int
	trackNum   int
	trackID    string
	artistID   string
	albumID    string
	isPlaying  bool
	isFavorite bool
	playCount  int

	num      *widget.RichText
	name     *widget.RichText
	artist   *CustomHyperlink
	album    *CustomHyperlink
	dur      *widget.RichText
	year     *widget.RichText
	favorite *fyne.Container
	rating   *StarRating
	bitrate  *widget.RichText
	plays    *widget.RichText
	size     *widget.RichText
	path     *widget.RichText

	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	playingIcon fyne.CanvasObject
}

func NewTrackRow(tracklist *Tracklist, playingIcon fyne.CanvasObject) *TrackRow {
	t := &TrackRow{tracklist: tracklist, playingIcon: playingIcon}
	t.ExtendBaseWidget(t)
	t.num = newTrailingAlignRichText()
	t.name = newTruncatingRichText()
	t.artist = NewCustomHyperlink()
	t.artist.OnTapped = func() { tracklist.onArtistTapped(t.artistID) }
	t.album = NewCustomHyperlink()
	t.album.OnTapped = func() { tracklist.onAlbumTapped(t.albumID) }
	t.dur = newTrailingAlignRichText()
	t.year = newTrailingAlignRichText()
	favorite := NewTappableIcon(myTheme.NotFavoriteIcon)
	favorite.OnTapped = t.toggleFavorited
	t.favorite = container.NewCenter(favorite)
	t.rating = NewStarRating()
	t.rating.StarSize = 16
	t.rating.OnRatingChanged = t.setTrackRating
	t.plays = newTrailingAlignRichText()
	t.bitrate = newTrailingAlignRichText()
	t.size = newTrailingAlignRichText()
	t.path = newTruncatingRichText()

	t.Content = container.New(tracklist.colLayout,
		t.num, t.name, t.artist, t.album, t.dur, t.year, t.favorite, t.rating, t.plays, t.bitrate, t.size, t.path)
	return t
}

func newTruncatingRichText() *widget.RichText {
	rt := widget.NewRichTextWithText("")
	rt.Wrapping = fyne.TextTruncate
	return rt
}

func newTrailingAlignRichText() *widget.RichText {
	rt := widget.NewRichTextWithText("")
	rt.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing
	return rt
}

func (t *TrackRow) Update(tm *trackModel, rowNum int) {
	t.Selected = tm.selected

	// Update info that can change if this row is bound to
	// a new track (*mediaprovider.Track)
	tr := tm.track
	if tr.ID != t.trackID {
		if t.Focused {
			fyne.CurrentApp().Driver().CanvasForObject(t).Focus(nil)
			t.Focused = false
		}
		t.trackID = tr.ID
		t.artistID = tr.ArtistIDs[0]
		t.albumID = tr.AlbumID

		t.name.Segments[0].(*widget.TextSegment).Text = tr.Name
		t.artist.SetText(tr.ArtistNames[0])
		t.artist.Disabled = tr.ArtistIDs[0] == ""
		t.album.SetText(tr.Album)
		t.dur.Segments[0].(*widget.TextSegment).Text = util.SecondsToTimeString(float64(tr.Duration))
		t.year.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(tr.Year)
		t.plays.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(int(tr.PlayCount))
		t.bitrate.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(tr.BitRate)
		t.size.Segments[0].(*widget.TextSegment).Text = util.BytesToSizeString(tr.Size)
		t.path.Segments[0].(*widget.TextSegment).Text = tr.FilePath
	}

	// Update track num if needed
	// (which can change based on bound *mediaprovider.Track or tracklist.AutoNumber)
	if t.trackNum != rowNum {
		discNum := -1
		var str string
		if rowNum < 0 {
			rowNum = tr.TrackNumber
			if t.tracklist.ShowDiscNumber {
				discNum = tr.DiscNumber
			}
		}
		t.trackNum = rowNum
		if discNum >= 0 {
			str = fmt.Sprintf("%d.%02d", discNum, rowNum)
		} else {
			str = strconv.Itoa(rowNum)
		}
		t.num.Segments[0].(*widget.TextSegment).Text = str
	}

	// Update play count if needed
	if tr.PlayCount != t.playCount {
		t.playCount = tr.PlayCount
		t.plays.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(int(tr.PlayCount))
	}

	// Render whether track is playing or not
	if isPlaying := t.tracklist.nowPlayingID == tr.ID; isPlaying != t.isPlaying {
		t.isPlaying = isPlaying
		t.name.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.dur.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.year.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.plays.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.bitrate.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.size.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
		t.path.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying

		if isPlaying {
			t.Content.(*fyne.Container).Objects[0] = container.NewCenter(t.playingIcon)
		} else {
			t.Content.(*fyne.Container).Objects[0] = t.num
		}
	}

	// Render favorite column
	if tr.Favorite {
		t.isFavorite = true
		t.favorite.Objects[0].(*TappableIcon).Resource = myTheme.FavoriteIcon
	} else {
		t.isFavorite = false
		t.favorite.Objects[0].(*TappableIcon).Resource = myTheme.NotFavoriteIcon
	}

	t.rating.Rating = tr.Rating

	// Show only columns configured to be visible
	t.artist.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnArtist)]
	t.album.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnAlbum)]
	t.dur.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnTime)]
	t.year.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnYear)]
	t.favorite.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnFavorite)]
	t.rating.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnRating)]
	t.plays.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnPlays)]
	t.bitrate.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnBitrate)]
	t.size.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnSize)]
	t.path.Hidden = !t.tracklist.visibleColumns[ColNumber(ColumnPath)]

	t.Refresh()
}

func (t *TrackRow) toggleFavorited() {
	if t.isFavorite {
		t.favorite.Objects[0].(*TappableIcon).Resource = myTheme.NotFavoriteIcon
		t.favorite.Refresh()
		t.isFavorite = false
		t.tracklist.onSetFavorite(t.trackID, false)
	} else {
		t.favorite.Objects[0].(*TappableIcon).Resource = myTheme.FavoriteIcon
		t.favorite.Refresh()
		t.isFavorite = true
		t.tracklist.onSetFavorite(t.trackID, true)
	}
}

func (t *TrackRow) setTrackRating(rating int) {
	t.tracklist.onSetRating(t.trackID, rating)
}

func (t *TrackRow) TappedSecondary(e *fyne.PointEvent) {
	if t.OnTappedSecondary != nil {
		t.OnTappedSecondary(e, t.trackIdx)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
