package browsing

import (
	"fmt"
	"log"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/player/mpv"
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
	statusLabel  *widget.Label
	nowPlayingID string
	container    *fyne.Container
}

type nowPlayingPageState struct {
	contr   *controller.Controller
	pool    *util.WidgetPool
	conf    *backend.NowPlayingPageConfig
	pm      *backend.PlaybackManager
	canRate bool
}

func NewNowPlayingPage(
	highlightedTrackID string,
	contr *controller.Controller,
	pool *util.WidgetPool,
	conf *backend.NowPlayingPageConfig,
	pm *backend.PlaybackManager,
	canRate bool,
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{
		contr: contr, pool: pool, conf: conf, pm: pm, canRate: canRate,
	}}
	a.ExtendBaseWidget(a)

	pm.OnPaused(a.formatStatusLine)
	pm.OnPlaying(a.formatStatusLine)
	pm.OnStopped(a.formatStatusLine)

	if t := a.pool.Obtain(util.WidgetTypeTracklist); t != nil {
		a.tracklist = t.(*widgets.Tracklist)
		a.tracklist.Reset()
	} else {
		a.tracklist = widgets.NewTracklist(nil)
	}
	a.tracklist.SetVisibleColumns(conf.TracklistColumns)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		a.conf.TracklistColumns = cols
	}
	remove := fyne.NewMenuItem("Remove from queue", a.onRemoveSelectedFromQueue)
	remove.Icon = theme.ContentRemoveIcon()
	a.tracklist.Options = widgets.TracklistOptions{
		AutoNumber:          true,
		DisablePlaybackMenu: true,
		DisableRating:       !canRate,
		AuxiliaryMenuItems: []*fyne.MenuItem{
			util.NewReorderTracksSubmenu(a.doSetNewTrackOrder),
			remove,
		},
	}
	contr.ConnectTracklistActions(a.tracklist)
	// override the default OnPlayTrackAt handler b/c we don't need to re-load the tracks into the queue
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.title = widget.NewRichTextWithText("Now Playing")
	a.title.Segments[0].(*widget.TextSegment).Style.SizeName = widget.RichTextStyleHeading.SizeName
	a.statusLabel = widget.NewLabel("Stopped")
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
	a.tracklist.Clear()
	a.pool.Release(util.WidgetTypeTracklist, a.tracklist)
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
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(song)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
}

var _ CanShowPlayTime = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnPlayTimeUpdate(_, _ float64) {
	a.formatStatusLine()
}

func (a *NowPlayingPage) formatStatusLine() {
	curPlayer := a.pm.CurrentPlayer()
	playerStats := curPlayer.GetStatus()
	lastStatus := a.statusLabel.Text
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

	mediaInfo := ""
	if state != "Stopped" {
		mediaInfo = a.formatMediaInfoStr(curPlayer)
	}
	if mediaInfo != "" {
		mediaInfo = " · " + mediaInfo
	}

	a.statusLabel.Text = fmt.Sprintf("%s%s | Total time: %s", status, mediaInfo, util.SecondsToTimeString(a.totalTime))
	if lastStatus != a.statusLabel.Text {
		a.statusLabel.Refresh()
	}
}

func (a *NowPlayingPage) formatMediaInfoStr(player player.BasePlayer) string {
	mpv, ok := player.(*mpv.Player)
	if !ok {
		return ""
	}
	audioInfo, err := mpv.GetMediaInfo()
	if err != nil {
		log.Printf("error getting playback status: %s", err.Error())
		return ""
	}
	codec := audioInfo.Codec
	if len(codec) <= 4 && !strings.EqualFold(codec, "opus") {
		codec = strings.ToUpper(codec) // FLAC, MP3, AAC, etc
	}

	// Note: bit depth intentionally omitted since MPV reports the decoded bit depth
	// i.e. 24 bit files get reported as 32 bit. Also b/c bit depth isn't meaningful for lossy.
	return fmt.Sprintf("%s %g kHz, %d kbps", codec, float64(audioInfo.Samplerate)/1000, audioInfo.Bitrate/1000)
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

func (a *NowPlayingPage) doSetNewTrackOrder(op sharedutil.TrackReorderOp) {
	// Since the tracklist view may be sorted in a different order than the
	// actual running order, we need to get the IDs of the selected tracks
	// from the tracklist and convert them to indices in the *original* run order
	ids := a.tracklist.SelectedTrackIDs()
	idxs := make([]int, 0, len(ids))
	for i, tr := range a.queue {
		if sharedutil.SliceContains(ids, tr.ID) {
			idxs = append(idxs, i)
		}
	}
	newTracks := sharedutil.ReorderTracks(a.queue, idxs, op)
	a.pm.UpdatePlayQueue(newTracks)

	// force-switch back to unsorted view to show new track order
	a.tracklist.SetSorting(widgets.TracklistSort{})
	a.tracklist.SetTracks(newTracks)
	a.tracklist.UnselectAll()
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
	return NewNowPlayingPage("", s.contr, s.pool, s.conf, s.pm, s.canRate)
}
