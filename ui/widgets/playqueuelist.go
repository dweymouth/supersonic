package widgets

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/os"
	"github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
)

type PlayQueueList struct {
	FocusList

	OnShowArtistPage func(artistID string)
	OnPlayTrackAt    func(idx int)

	nowPlayingID string
	colLayout    *layouts.ColumnsLayout

	tracksMutex sync.Mutex
	tracks      []*util.TrackListModel
}

func (t *PlayQueueList) onArtistTapped(artistID string) {
	if t.OnShowArtistPage != nil {
		t.OnShowArtistPage(artistID)
	}
}

func (p *PlayQueueList) onPlayTrackAt(idx int) {
	if p.OnPlayTrackAt != nil {
		p.OnPlayTrackAt(idx)
	}
}

func (p *PlayQueueList) onSelectTrack(idx int) {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		mod := d.CurrentKeyModifiers()
		if mod&os.ControlModifier != 0 {
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
	p.tracksMutex.Lock()
	defer p.tracksMutex.Unlock()
	util.SelectTrack(p.tracks, idx)
}

func (p *PlayQueueList) selectAddOrRemove(idx int) {
	p.tracksMutex.Lock()
	defer p.tracksMutex.Unlock()
	p.tracks[idx].Selected = !p.tracks[idx].Selected
}

func (p *PlayQueueList) selectRange(idx int) {
	p.tracksMutex.Lock()
	defer p.tracksMutex.Unlock()
	util.SelectTrackRange(p.tracks, idx)
}

type PlayQueueListRow struct {
	FocusListRowBase

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

func NewPlayQueueListRow(playQueueList *PlayQueueList, playingIcon fyne.CanvasObject) *PlayQueueListRow {
	p := &PlayQueueListRow{
		playingIcon:   playingIcon,
		playQueueList: playQueueList,
		num:           widget.NewLabel(""),
		cover:         NewImagePlaceholder(theme.TracksIcon, 64),
		title:         util.NewTruncatingLabel(),
		artist:        NewMultiHyperlink(),
		time:          util.NewTrailingAlignLabel(),
	}
	p.artist.OnTapped = playQueueList.onArtistTapped
	p.OnDoubleTapped = func() {
		playQueueList.onPlayTrackAt(p.ItemID())
	}
	p.OnTapped = func() {
		playQueueList.onSelectTrack(p.ItemID())
	}
	//p.title.TextStyle.Bold = true
	p.ExtendBaseWidget(p)
	p.Content = container.New(playQueueList.colLayout,
		container.NewCenter(p.num),
		p.cover,
		container.New(&layouts.VboxCustomPadding{ExtraPad: -15},
			p.title, p.artist),
		container.NewCenter(p.time),
	)
	return p
}

func (p *PlayQueueListRow) Update(tm *util.TrackListModel, rowNum int) {
	changed := false
	if tm.Selected != p.Selected {
		p.Selected = tm.Selected
		changed = true
	}

	// Update info that can change if this row is bound to
	// a new track (*mediaprovider.Track)
	tr := tm.Track
	if tr.ID != p.trackID {
		p.EnsureUnfocused()
		p.trackID = tr.ID
		p.artist.BuildSegments(tr.ArtistNames, tr.ArtistIDs)
		p.time.Text = util.SecondsToTimeString(float64(tr.Duration))
		changed = true
	}

	// Render whether track is playing or not
	if isPlaying := p.playQueueList.nowPlayingID == tr.ID; isPlaying != p.isPlaying {
		p.isPlaying = isPlaying
		p.title.TextStyle.Bold = isPlaying

		if isPlaying {
			p.Content.(*fyne.Container).Objects[0] = p.playingIcon
		} else {
			p.Content.(*fyne.Container).Objects[0] = p.num
		}
		changed = true
	}

	if changed {
		p.Refresh()
	}
}
