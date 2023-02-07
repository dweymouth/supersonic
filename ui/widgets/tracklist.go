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
	"github.com/dweymouth/go-subsonic"
)

type TrackRow struct {
	widget.BaseWidget

	trackID       string
	prevTrackID   string
	prevIsPlaying bool

	tappedAt int64 // unixMillis

	num    *widget.RichText
	name   *widget.RichText
	artist *widget.RichText
	dur    *widget.RichText

	OnTapped       func()
	OnDoubleTapped func()

	selectionRect *canvas.Rectangle

	container *fyne.Container
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
	if tr.ID == t.prevTrackID && isPlaying == t.prevIsPlaying {
		return
	}
	t.prevTrackID = t.trackID
	t.prevIsPlaying = isPlaying
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

type Tracklist struct {
	widget.BaseWidget

	Tracks        []*subsonic.Child
	AutoNumber    bool
	OnPlayTrackAt func(int)
	SelectionMgr  util.ListSelectionManager

	nowPlayingIdx int
	colLayout     *layouts.ColumnsLayout
	hdr           *ListHeader
	list          *widget.List
	container     *fyne.Container
}

func NewTracklist(tracks []*subsonic.Child) *Tracklist {
	t := &Tracklist{Tracks: tracks, nowPlayingIdx: -1}
	t.ExtendBaseWidget(t)
	t.SelectionMgr = util.NewListSelectionManager(func() int { return len(t.Tracks) })
	t.colLayout = layouts.NewColumnsLayout([]float32{35, -1, -1, 60})
	t.hdr = NewListHeader([]ListColumn{{"#", true}, {"Title", false}, {"Artist", false}, {"Time", true}}, t.colLayout)
	t.list = widget.NewList(
		func() int { return len(t.Tracks) },
		func() fyne.CanvasObject { return NewTrackRow(t.colLayout) },
		func(itemID widget.ListItemID, item fyne.CanvasObject) {
			tr := item.(*TrackRow)
			tr.OnTapped = func() { t.onSelectTrack(itemID) }
			tr.OnDoubleTapped = func() { t.onPlayTrackAt(itemID) }
			tr.selectionRect.Hidden = !t.SelectionMgr.IsSelected(itemID)
			i := itemID + 1
			if !t.AutoNumber {
				i = -1 // signal that we want to use the track num.
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
			t.SelectionMgr.SelectAddOrRemove(idx)
		} else if (d.ActiveKeyModifiers() & fyne.KeyModifierShift) != 0 {
			t.SelectionMgr.SelectRange(idx)
		} else {
			t.SelectionMgr.Select(idx)
		}
	} else {
		t.SelectionMgr.Select(idx)
	}
	t.list.Refresh()
}
