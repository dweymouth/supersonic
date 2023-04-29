package widgets

import (
	"fmt"
	"log"
	"strconv"
	"supersonic/res"
	"supersonic/sharedutil"
	"supersonic/ui/layouts"
	"supersonic/ui/os"
	"supersonic/ui/util"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

const (
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

type Tracklist struct {
	widget.BaseWidget

	// Tracks is the set of tracks displayed by the widget.
	// Direct access to this is not thread-safe but OK for
	// views that only load tracks into the widget once at page load.
	Tracks []*subsonic.Child

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

	// user action callbacks
	OnPlayTrackAt   func(int)
	OnPlaySelection func(tracks []*subsonic.Child)
	OnAddToQueue    func(trackIDs []*subsonic.Child)
	OnAddToPlaylist func(trackIDs []string)
	OnSetFavorite   func(trackIDs []string, fav bool)
	OnSetRating     func(trackIDs []string, rating int)

	OnShowArtistPage func(artistID string)
	OnShowAlbumPage  func(albumID string)

	OnColumnVisibilityMenuShown func(*widget.PopUp)
	OnVisibleColumnsChanged     func([]string)
	OnTrackShown                func(tracknum int)

	visibleColumns []bool

	tracksMutex  sync.RWMutex
	selectionMgr util.ListSelectionManager
	nowPlayingID string
	colLayout    *layouts.ColumnsLayout
	hdr          *ListHeader
	list         *widget.List
	ctxMenu      *fyne.Menu
	container    *fyne.Container
}

func NewTracklist(tracks []*subsonic.Child) *Tracklist {
	t := &Tracklist{Tracks: tracks, visibleColumns: make([]bool, 12)}

	t.ExtendBaseWidget(t)
	t.selectionMgr = util.NewListSelectionManager(t.lenTracks)
	// #, Title, Artist, Album, Time, Year, Favorite, Rating, Plays, Bitrate, Size, Path
	t.colLayout = layouts.NewColumnsLayout([]float32{40, -1, -1, -1, 50, 50, 45, 95, 55, 65, 70, -1})
	t.buildHeader()
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
			tr.Selected = t.selectionMgr.IsSelected(itemID)
			i := -1 // signal that we want to display the actual track num.
			if t.AutoNumber {
				i = itemID + 1
			}
			tr.Update(t.TrackAt(itemID), i)
			if t.OnTrackShown != nil {
				t.OnTrackShown(itemID)
			}
		})
	t.container = container.NewBorder(t.hdr, nil, nil, nil, t.list)
	return t
}

func (t *Tracklist) buildHeader() {
	t.hdr = NewListHeader([]ListColumn{
		{Text: "#", AlignTrailing: true, CanToggleVisible: false},
		{Text: "Title", AlignTrailing: false, CanToggleVisible: false},
		{Text: "Artist", AlignTrailing: false, CanToggleVisible: true},
		{Text: "Album", AlignTrailing: false, CanToggleVisible: true},
		{Text: "Time", AlignTrailing: true, CanToggleVisible: true},
		{Text: "Year", AlignTrailing: true, CanToggleVisible: true},
		{Text: "Fav.", AlignTrailing: false, CanToggleVisible: true},
		{Text: "Rating", AlignTrailing: false, CanToggleVisible: true},
		{Text: "Plays", AlignTrailing: true, CanToggleVisible: true},
		{Text: "Bitrate", AlignTrailing: true, CanToggleVisible: true},
		{Text: "Size", AlignTrailing: true, CanToggleVisible: true},
		{Text: "File Path", AlignTrailing: false, CanToggleVisible: true}},
		t.colLayout)
}

// Gets the track at the given index. Thread-safe.
func (t *Tracklist) TrackAt(idx int) *subsonic.Child {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	if idx >= len(t.Tracks) {
		return nil
	}
	return t.Tracks[idx]
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

func (t *Tracklist) SetNowPlaying(trackID string) {
	t.nowPlayingID = trackID
	t.list.Refresh()
}

func (t *Tracklist) IncrementPlayCount(trackID string) {
	t.tracksMutex.RLock()
	tr := sharedutil.FindTrackByID(trackID, t.Tracks)
	t.tracksMutex.RUnlock()
	if tr != nil {
		tr.PlayCount += 1
		t.list.Refresh()
	}
}

// Remove all tracks from the tracklist. Thread-safe.
func (t *Tracklist) Clear() {
	t.selectionMgr.UnselectAll()
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.Tracks = nil
}

// Append more tracks to the tracklist. Thread-safe.
func (t *Tracklist) AppendTracks(trs []*subsonic.Child) {
	t.tracksMutex.Lock()
	defer t.tracksMutex.Unlock()
	t.Tracks = append(t.Tracks, trs...)
}

func (t *Tracklist) SelectAll() {
	t.selectionMgr.SelectAll()
	t.list.Refresh()
}

func (t *Tracklist) UnselectAll() {
	t.selectionMgr.UnselectAll()
	t.list.Refresh()
}

func (t *Tracklist) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
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
			t.selectionMgr.SelectAddOrRemove(idx)
		} else if mod&fyne.KeyModifierShift != 0 {
			t.selectionMgr.SelectRange(idx)
		} else {
			t.selectionMgr.Select(idx)
		}
	} else {
		t.selectionMgr.Select(idx)
	}
	t.list.Refresh()
}

func (t *Tracklist) onShowContextMenu(e *fyne.PointEvent, trackIdx int) {
	t.selectionMgr.Select(trackIdx)
	t.list.Refresh()
	if t.ctxMenu == nil {
		t.ctxMenu = fyne.NewMenu("")
		if !t.DisablePlaybackMenu {
			t.ctxMenu.Items = append(t.ctxMenu.Items,
				fyne.NewMenuItem("Play", func() {
					if t.OnPlaySelection != nil {
						t.OnPlaySelection(t.selectedTracks())
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
					t.OnAddToPlaylist(t.selectedTrackIDs())
				}
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
		if len(t.AuxiliaryMenuItems) > 0 {
			t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
			t.ctxMenu.Items = append(t.ctxMenu.Items, t.AuxiliaryMenuItems...)
		}
	}
	widget.ShowPopUpMenuAtPosition(t.ctxMenu, fyne.CurrentApp().Driver().CanvasForObject(t), e.AbsolutePosition)
}

func (t *Tracklist) onSetFavorite(trackID string, fav bool) {
	t.tracksMutex.RLock()
	tr := sharedutil.FindTrackByID(trackID, t.Tracks)
	t.tracksMutex.RUnlock()
	t.onSetFavorites([]*subsonic.Child{tr}, fav, false)
}

func (t *Tracklist) onSetFavorites(tracks []*subsonic.Child, fav bool, needRefresh bool) {
	for _, tr := range tracks {
		if fav {
			tr.Starred = time.Now()
		} else {
			tr.Starred = time.Time{}
		}
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
	tr := sharedutil.FindTrackByID(trackID, t.Tracks)
	t.tracksMutex.RUnlock()
	tr.UserRating = rating
	// notify listener
	if t.OnSetRating != nil {
		t.OnSetRating([]string{trackID}, rating)
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

func (t *Tracklist) selectedTracks() []*subsonic.Child {
	sel := t.selectionMgr.GetSelection()
	tracks := make([]*subsonic.Child, 0, len(sel))
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	for _, idx := range sel {
		tracks = append(tracks, t.Tracks[idx])
	}
	return tracks
}

func (t *Tracklist) selectedTrackIDs() []string {
	sel := t.selectionMgr.GetSelection()
	tracks := make([]string, 0, len(sel))
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	for _, idx := range sel {
		tracks = append(tracks, t.Tracks[idx].ID)
	}
	return tracks
}

func (t *Tracklist) SelectedTrackIndexes() []int {
	return t.selectionMgr.GetSelection()
}

func (t *Tracklist) lenTracks() int {
	t.tracksMutex.RLock()
	defer t.tracksMutex.RUnlock()
	return len(t.Tracks)
}

func ColNumber(colName string) int {
	// built-in columns # and Title are always visible
	switch colName {
	case ColumnArtist:
		return 2
	case ColumnAlbum:
		return 3
	case ColumnTime:
		return 4
	case ColumnYear:
		return 5
	case ColumnFavorite:
		return 6
	case ColumnRating:
		return 7
	case ColumnPlays:
		return 8
	case ColumnBitrate:
		return 9
	case ColumnSize:
		return 10
	case ColumnPath:
		return 11
	default:
		log.Printf("error: Tracklist: invalid column name %s", colName)
		return -100
	}
}

func colName(i int) string {
	// built-in columns # and Title are always visible
	switch i {
	case 2:
		return ColumnArtist
	case 3:
		return ColumnAlbum
	case 4:
		return ColumnTime
	case 5:
		return ColumnYear
	case 6:
		return ColumnFavorite
	case 7:
		return ColumnRating
	case 8:
		return ColumnPlays
	case 9:
		return ColumnBitrate
	case 10:
		return ColumnSize
	case 11:
		return ColumnPath
	default:
		return ""
	}
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
	playCount  int64

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
	favorite := NewTappbaleIcon(res.ResHeartOutlineInvertPng)
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

func (t *TrackRow) Update(tr *subsonic.Child, rowNum int) {
	// Update info that can change if this row is bound to
	// a new track (*subsonic.Child)
	if tr.ID != t.trackID {
		if t.Focused {
			fyne.CurrentApp().Driver().CanvasForObject(t).Focus(nil)
			t.Focused = false
		}
		t.trackID = tr.ID
		t.artistID = tr.ArtistID
		t.albumID = tr.AlbumID

		t.name.Segments[0].(*widget.TextSegment).Text = tr.Title
		t.artist.SetText(tr.Artist)
		t.artist.Disabled = tr.ArtistID == ""
		t.album.SetText(tr.Album)
		t.dur.Segments[0].(*widget.TextSegment).Text = util.SecondsToTimeString(float64(tr.Duration))
		t.year.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(tr.Year)
		t.plays.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(int(tr.PlayCount))
		t.bitrate.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(tr.BitRate)
		t.size.Segments[0].(*widget.TextSegment).Text = util.BytesToSizeString(tr.Size)
		t.path.Segments[0].(*widget.TextSegment).Text = tr.Path
	}

	// Update track num if needed
	// (which can change based on bound *subsonic.Child or tracklist.AutoNumber)
	if t.trackNum != rowNum {
		discNum := -1
		var str string
		if rowNum < 0 {
			rowNum = tr.Track
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
	// TODO: right now the only way for the favorite status to change while the tracklist is visible
	// is by the user clicking on the heart icon in the favorites column
	// If this changes in the future (e.g. context menu entry on tracklist), then we will
	// need better state management/onChanged notif so we know to re-render the column
	// (maybe update the Starred field directly on the track struct and issue a Refresh call -
	// like we do to update the now playing value when scrobbles happen)
	if tr.Starred.IsZero() {
		t.isFavorite = false
		t.favorite.Objects[0].(*TappableIcon).Resource = res.ResHeartOutlineInvertPng
	} else {
		t.isFavorite = true
		t.favorite.Objects[0].(*TappableIcon).Resource = res.ResHeartFilledInvertPng
	}

	t.rating.Rating = tr.UserRating

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
		t.favorite.Objects[0].(*TappableIcon).Resource = res.ResHeartOutlineInvertPng
		t.favorite.Refresh()
		t.isFavorite = false
		t.tracklist.onSetFavorite(t.trackID, false)
	} else {
		t.favorite.Objects[0].(*TappableIcon).Resource = res.ResHeartFilledInvertPng
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
