package browsing

import (
	"fmt"
	"log"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NowPlayingPage struct {
	widget.BaseWidget

	nowPlayingPageState

	queue     []*mediaprovider.Track
	totalTime float64

	title        *widget.RichText
	tracklist    *widgets.Tracklist
	statusLabel  *widget.RichText
	nowPlayingID string
	container    *fyne.Container
}

type nowPlayingPageState struct {
	contr *controller.Controller
	conf  *backend.NowPlayingPageConfig
	pm    *backend.PlaybackManager
	p     *player.Player
}

func NewNowPlayingPage(
	highlightedTrackID string,
	contr *controller.Controller,
	conf *backend.NowPlayingPageConfig,
	pm *backend.PlaybackManager,
	p *player.Player, // TODO: once other player backends are supported (eg uPnP), refactor
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{contr: contr, conf: conf, pm: pm, p: p}}
	a.ExtendBaseWidget(a)

	p.OnPaused(a.formatStatusLine)
	p.OnPlaying(a.formatStatusLine)
	p.OnStopped(a.formatStatusLine)

	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns(conf.TracklistColumns)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		a.conf.TracklistColumns = cols
	}
	a.tracklist.AutoNumber = true
	a.tracklist.DisablePlaybackMenu = true
	contr.ConnectTracklistActions(a.tracklist)
	// override the default OnPlayTrackAt handler b/c we don't need to re-load the tracks into the queue
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.tracklist.AuxiliaryMenuItems = []*fyne.MenuItem{
		fyne.NewMenuItem("Remove from queue", a.onRemoveSelectedFromQueue),
	}
	a.title = widget.NewRichTextWithText("Now Playing")
	a.title.Segments[0].(*widget.TextSegment).Style.SizeName = widget.RichTextStyleHeading.SizeName
	a.statusLabel = widget.NewRichTextWithText("Stopped")
	statusLabelCtr := container.New(&layouts.VboxCustomPadding{ExtraPad: -5},
		myTheme.NewThemedRectangle(theme.ColorNameInputBorder),
		a.statusLabel,
	)
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(a.title, statusLabelCtr, nil, nil, a.tracklist))
	a.load(highlightedTrackID)
	return a
}

func (a *NowPlayingPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *NowPlayingPage) Save() SavedPage {
	nps := a.nowPlayingPageState
	return &nps
}

func (a *NowPlayingPage) Route() controller.Route {
	return controller.NowPlayingRoute("")
}

func (a *NowPlayingPage) Tapped(*fyne.PointEvent) {
	a.tracklist.UnselectAll()
}

func (a *NowPlayingPage) SelectAll() {
	a.tracklist.SelectAll()
}

var _ CanShowNowPlaying = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnSongChange(song, lastScrobbledIfAny *mediaprovider.Track) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
}

var _ CanShowPlayTime = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnPlayTimeUpdate(_, _ float64) {
	a.formatStatusLine()
}

func (a *NowPlayingPage) formatStatusLine() {
	playerStats := a.p.GetStatus()
	ts := a.statusLabel.Segments[0].(*widget.TextSegment)
	lastStatus := ts.Text
	state := "Stopped"
	switch playerStats.State {
	case player.Paused:
		state = "Paused"
	case player.Playing:
		state = "Playing"
	}

	dur := 0.0
	if np := a.pm.NowPlaying(); np != nil {
		dur = float64(np.Duration)
	}
	statusSuffix := ""
	trackNum := 0
	if state != "Stopped" {
		trackNum = a.pm.NowPlayingIndex() + 1
		statusSuffix = fmt.Sprintf(" %s/%s",
			util.SecondsToTimeString(playerStats.TimePos),
			util.SecondsToTimeString(dur))
	}
	status := fmt.Sprintf("%s (%d/%d)%s", state, trackNum,
		len(a.queue), statusSuffix)

	if state == "Stopped" {
		ts.Text = fmt.Sprintf("%s | Total time: %s", status, util.SecondsToTimeString(a.totalTime))
	} else {
		audioInfo, err := a.p.GetMediaInfo()
		if err != nil {
			log.Printf("error getting playback status: %s", err.Error())
		}
		codec := audioInfo.Codec
		if len(codec) <= 4 && !strings.EqualFold(codec, "opus") {
			codec = strings.ToUpper(codec) // FLAC, MP3, AAC, etc
		}

		// Note: bit depth intentionally omitted since MPV reports the decoded bit depth
		// i.e. 24 bit files get reported as 32 bit. Also b/c bit depth isn't meaningful for lossy.
		ts.Text = fmt.Sprintf("%s · %s %g kHz, %d kbps | Total time: %s",
			status,
			codec,
			float64(audioInfo.Samplerate)/1000,
			audioInfo.Bitrate/1000,
			util.SecondsToTimeString(a.totalTime))
	}
	if lastStatus != ts.Text {
		a.statusLabel.Refresh()
	}
}

func (a *NowPlayingPage) Reload() {
	a.load("")
}

func (a *NowPlayingPage) onPlayTrackAt(tracknum int) {
	_ = a.pm.PlayTrackAt(tracknum)
}

func (a *NowPlayingPage) onRemoveSelectedFromQueue() {
	a.pm.RemoveTracksFromQueue(a.tracklist.SelectedTrackIDs())
	a.tracklist.UnselectAll()
	a.Reload()
}

// does not make calls to server - can safely be run in UI callbacks
func (a *NowPlayingPage) load(highlightedTrackID string) {
	a.queue = a.pm.GetPlayQueue()
	a.tracklist.SetTracks(a.queue)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	if highlightedTrackID != "" {
		a.tracklist.SelectAndScrollToTrack(highlightedTrackID)
	}
	a.totalTime = 0.0
	for _, tr := range a.queue {
		a.totalTime += float64(tr.Duration)
	}
	a.formatStatusLine()
}

func (s *nowPlayingPageState) Restore() Page {
	return NewNowPlayingPage("", s.contr, s.conf, s.pm, s.p)
}
