package widgets

import (
	"strconv"
	"supersonic/ui/layouts"
	"supersonic/ui/os"
	"supersonic/ui/util"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type TracklistColumn string

const (
	ColumnArtist  TracklistColumn = "Artist"
	ColumnAlbum   TracklistColumn = "Album"
	ColumnTime    TracklistColumn = "Time"
	ColumnPlays   TracklistColumn = "Plays"
	ColumnBitrate TracklistColumn = "Bitrate"
)

type Tracklist struct {
	widget.BaseWidget

	Tracks     []*subsonic.Child
	AutoNumber bool
	// must be set before the context menu is shown for the first time
	AuxiliaryMenuItems  []*fyne.MenuItem
	DisablePlaybackMenu bool

	// user action callbacks
	OnPlayTrackAt   func(int)
	OnPlaySelection func(tracks []*subsonic.Child)
	OnAddToQueue    func(trackIDs []*subsonic.Child)
	OnAddToPlaylist func(trackIDs []string)

	visibleColumns []bool

	selectionMgr  util.ListSelectionManager
	nowPlayingIdx int
	colLayout     *layouts.ColumnsLayout
	hdr           *ListHeader
	list          *widget.List
	ctxMenu       *fyne.Menu
	container     *fyne.Container
}

func NewTracklist(tracks []*subsonic.Child) *Tracklist {
	t := &Tracklist{Tracks: tracks, nowPlayingIdx: -1, visibleColumns: make([]bool, 5)}

	t.ExtendBaseWidget(t)
	t.selectionMgr = util.NewListSelectionManager(func() int { return len(t.Tracks) })
	t.colLayout = layouts.NewColumnsLayout([]float32{35, -1, -1, -1, 60, 65, 75})
	t.hdr = NewListHeader([]ListColumn{
		{"#", true}, {"Title", false}, {"Artist", false}, {"Album", false}, {"Time", true}, {"Plays", true}, {"Bitrate", true}},
		t.colLayout)
	playingIcon := container.NewCenter(container.NewHBox(NewHSpace(2), widget.NewIcon(theme.MediaPlayIcon())))
	t.list = widget.NewList(
		func() int { return len(t.Tracks) },
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
			tr.selectionRect.Hidden = !t.selectionMgr.IsSelected(itemID)
			i := -1 // signal that we want to display the actual track num.
			if t.AutoNumber {
				i = itemID + 1
			}
			tr.Update(t.Tracks[itemID], itemID == t.nowPlayingIdx, i)
		})
	t.container = container.NewBorder(t.hdr, nil, nil, nil, t.list)
	return t
}

func (t *Tracklist) SetVisibleColumns(cols []TracklistColumn) {
	for i := range t.visibleColumns {
		t.visibleColumns[i] = false
	}
	for _, col := range cols {
		t.visibleColumns[col.ColNumber()] = true
	}
}

func (t *Tracklist) SetNowPlaying(trackID string) {
	t.nowPlayingIdx = -1
	for i, tr := range t.Tracks {
		if tr.ID == trackID {
			t.nowPlayingIdx = i
			break
		}
	}
	t.list.Refresh()
}

func (t *Tracklist) SelectAll() {
	t.selectionMgr.SelectAll()
	t.Refresh()
}

func (t *Tracklist) UnselectAll() {
	t.selectionMgr.UnselectAll()
	t.Refresh()
}

func (t *Tracklist) Refresh() {
	for i, tf := range t.visibleColumns {
		// first 2 columns are built-in and always visible
		t.hdr.SetColumnVisible(i+2, tf)
	}
	t.BaseWidget.Refresh()
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
		if d.ActiveKeyModifiers()&os.ControlModifier != 0 {
			t.selectionMgr.SelectAddOrRemove(idx)
		} else if (d.ActiveKeyModifiers() & fyne.KeyModifierShift) != 0 {
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
	t.Refresh()
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
		if len(t.AuxiliaryMenuItems) > 0 {
			t.ctxMenu.Items = append(t.ctxMenu.Items, fyne.NewMenuItemSeparator())
			t.ctxMenu.Items = append(t.ctxMenu.Items, t.AuxiliaryMenuItems...)
		}
	}
	widget.ShowPopUpMenuAtPosition(t.ctxMenu, fyne.CurrentApp().Driver().CanvasForObject(t), e.AbsolutePosition)
}

func (t *Tracklist) selectedTracks() []*subsonic.Child {
	sel := t.selectionMgr.GetSelection()
	tracks := make([]*subsonic.Child, 0, len(sel))
	for _, idx := range sel {
		tracks = append(tracks, t.Tracks[idx])
	}
	return tracks
}

func (t *Tracklist) selectedTrackIDs() []string {
	sel := t.selectionMgr.GetSelection()
	tracks := make([]string, 0, len(sel))
	for _, idx := range sel {
		tracks = append(tracks, t.Tracks[idx].ID)
	}
	return tracks
}

func (t *Tracklist) SelectedTrackIndexes() []int {
	return t.selectionMgr.GetSelection()
}

func (c TracklistColumn) ColNumber() int {
	// built-in columns # and Title are always visible
	switch c {
	case ColumnArtist:
		return 0
	case ColumnAlbum:
		return 1
	case ColumnTime:
		return 2
	case ColumnPlays:
		return 3
	case ColumnBitrate:
		return 4
	default:
		return -100
	}
}

type TrackRow struct {
	widget.BaseWidget

	// internal state
	tracklist *Tracklist
	trackIdx  int
	trackID   string
	isPlaying bool
	tappedAt  int64 // unixMillis

	num     *widget.RichText
	name    *widget.RichText
	artist  *widget.RichText
	album   *widget.RichText
	dur     *widget.RichText
	bitrate *widget.RichText
	plays   *widget.RichText

	OnTapped          func()
	OnDoubleTapped    func()
	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	playingIcon   fyne.CanvasObject
	selectionRect *canvas.Rectangle
	container     *fyne.Container
}

func NewTrackRow(tracklist *Tracklist, playingIcon fyne.CanvasObject) *TrackRow {
	t := &TrackRow{tracklist: tracklist, playingIcon: playingIcon}
	t.ExtendBaseWidget(t)
	t.num = widget.NewRichTextWithText("")
	t.num.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing
	t.name = widget.NewRichTextWithText("")
	t.name.Wrapping = fyne.TextTruncate
	t.artist = widget.NewRichTextWithText("")
	t.artist.Wrapping = fyne.TextTruncate
	t.album = widget.NewRichTextWithText("")
	t.album.Wrapping = fyne.TextTruncate
	t.dur = widget.NewRichTextWithText("")
	t.dur.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing
	t.plays = widget.NewRichTextWithText("")
	t.plays.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing
	t.bitrate = widget.NewRichTextWithText("")
	t.bitrate.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing

	t.selectionRect = canvas.NewRectangle(theme.SelectionColor())
	t.selectionRect.Hidden = true
	t.container = container.NewMax(t.selectionRect,
		container.New(tracklist.colLayout,
			t.num, t.name, t.artist, t.album, t.dur, t.plays, t.bitrate))
	return t
}

func (t *TrackRow) Update(tr *subsonic.Child, isPlaying bool, rowNum int) {
	if tr.ID == t.trackID && isPlaying == t.isPlaying {
		return
	}
	t.isPlaying = isPlaying
	t.trackID = tr.ID

	if rowNum < 0 {
		rowNum = tr.Track
	}
	t.num.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(rowNum)
	t.name.Segments[0].(*widget.TextSegment).Text = tr.Title
	t.artist.Segments[0].(*widget.TextSegment).Text = tr.Artist
	t.album.Segments[0].(*widget.TextSegment).Text = tr.Album
	t.dur.Segments[0].(*widget.TextSegment).Text = util.SecondsToTimeString(float64(tr.Duration))
	t.plays.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(int(tr.PlayCount))
	t.bitrate.Segments[0].(*widget.TextSegment).Text = strconv.Itoa(tr.BitRate)

	t.name.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
	t.artist.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
	t.album.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
	t.dur.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
	t.plays.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying
	t.bitrate.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = isPlaying

	t.artist.Hidden = !t.tracklist.visibleColumns[ColumnArtist.ColNumber()]
	t.album.Hidden = !t.tracklist.visibleColumns[ColumnAlbum.ColNumber()]
	t.dur.Hidden = !t.tracklist.visibleColumns[ColumnTime.ColNumber()]
	t.plays.Hidden = !t.tracklist.visibleColumns[ColumnPlays.ColNumber()]
	t.bitrate.Hidden = !t.tracklist.visibleColumns[ColumnBitrate.ColNumber()]

	if isPlaying {
		t.container.Objects[1].(*fyne.Container).Objects[0] = container.NewCenter(t.playingIcon)
	} else {
		t.container.Objects[1].(*fyne.Container).Objects[0] = t.num
	}

	t.Refresh()
}

func (t *TrackRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.container)
}

// We implement our own double tapping so that the Tapped behavior
// can be triggered instantly.
func (t *TrackRow) Tapped(*fyne.PointEvent) {
	prevTap := t.tappedAt
	t.tappedAt = time.Now().UnixMilli()
	if t.tappedAt-prevTap < 300 {
		if t.OnDoubleTapped != nil {
			t.OnDoubleTapped()
		}
	} else {
		if t.OnTapped != nil {
			t.OnTapped()
		}
	}
}

func (t *TrackRow) TappedSecondary(e *fyne.PointEvent) {
	if t.OnTappedSecondary != nil {
		t.OnTappedSecondary(e, t.trackIdx)
	}
}
