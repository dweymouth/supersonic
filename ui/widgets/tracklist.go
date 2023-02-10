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

type TrackRow struct {
	widget.BaseWidget

	// internal state
	trackIdx  int
	trackID   string
	isPlaying bool
	tappedAt  int64 // unixMillis

	num    *widget.RichText
	name   *widget.RichText
	artist *widget.RichText
	dur    *widget.RichText

	OnTapped          func()
	OnDoubleTapped    func()
	OnTappedSecondary func(e *fyne.PointEvent, trackIdx int)

	selectionRect *canvas.Rectangle
	container     *fyne.Container
}

func NewTrackRow(layout *layouts.ColumnsLayout) *TrackRow {
	t := &TrackRow{}
	t.ExtendBaseWidget(t)
	t.num = widget.NewRichTextWithText("")
	t.num.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing
	t.name = widget.NewRichTextWithText("")
	t.name.Wrapping = fyne.TextTruncate
	t.artist = widget.NewRichTextWithText("")
	t.artist.Wrapping = fyne.TextTruncate
	t.dur = widget.NewRichTextWithText("")
	t.dur.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignTrailing

	t.selectionRect = canvas.NewRectangle(theme.SelectionColor())
	t.selectionRect.Hidden = true
	t.container = container.NewMax(t.selectionRect,
		container.New(layout,
			t.num, t.name, t.artist, t.dur))
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
	t.dur.Segments[0].(*widget.TextSegment).Text = util.SecondsToTimeString(float64(tr.Duration))

	t.num.Segments[0].(*widget.TextSegment).Style.TextStyle = fyne.TextStyle{Bold: isPlaying}
	t.name.Segments[0].(*widget.TextSegment).Style.TextStyle = fyne.TextStyle{Bold: isPlaying, Italic: isPlaying}
	t.artist.Segments[0].(*widget.TextSegment).Style.TextStyle = fyne.TextStyle{Bold: isPlaying, Italic: isPlaying}
	t.dur.Segments[0].(*widget.TextSegment).Style.TextStyle = fyne.TextStyle{Bold: isPlaying}

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

type Tracklist struct {
	widget.BaseWidget

	Tracks     []*subsonic.Child
	AutoNumber bool

	// user action callbacks
	OnPlayTrackAt   func(int)
	OnPlaySelection func(tracks []*subsonic.Child)
	OnAddToQueue    func(trackIDs []*subsonic.Child)
	OnAddToPlaylist func(trackIDs []*subsonic.Child)

	selectionMgr  util.ListSelectionManager
	nowPlayingIdx int
	colLayout     *layouts.ColumnsLayout
	hdr           *ListHeader
	list          *widget.List
	ctxMenu       *fyne.Menu
	container     *fyne.Container
}

func NewTracklist(tracks []*subsonic.Child) *Tracklist {
	t := &Tracklist{Tracks: tracks, nowPlayingIdx: -1}
	t.ExtendBaseWidget(t)
	t.selectionMgr = util.NewListSelectionManager(func() int { return len(t.Tracks) })
	t.colLayout = layouts.NewColumnsLayout([]float32{35, -1, -1, 60})
	t.hdr = NewListHeader([]ListColumn{{"#", true}, {"Title", false}, {"Artist", false}, {"Time", true}}, t.colLayout)
	t.list = widget.NewList(
		func() int { return len(t.Tracks) },
		func() fyne.CanvasObject {
			tr := NewTrackRow(t.colLayout)
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

func (t *Tracklist) UnselectAll() {
	t.selectionMgr.UnselectAll()
	t.Refresh()
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
		t.ctxMenu = fyne.NewMenu("",
			fyne.NewMenuItem("Play", func() {
				if t.OnPlaySelection != nil {
					t.OnPlaySelection(t.selectedTracks())
				}
			}),
			fyne.NewMenuItem("Add to queue", func() {
				if t.OnPlaySelection != nil {
					t.OnAddToQueue(t.selectedTracks())
				}
			}),
			//fyne.NewMenuItem("Add to playlist...", func() {}),
		)
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
