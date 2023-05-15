package browsing

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type NowPlayingPage struct {
	widget.BaseWidget

	nowPlayingPageState

	title        *widget.RichText
	tracklist    *widgets.Tracklist
	nowPlayingID string
	container    *fyne.Container
}

type nowPlayingPageState struct {
	contr *controller.Controller
	conf  *backend.NowPlayingPageConfig
	sm    *backend.ServerManager
	pm    *backend.PlaybackManager
}

func NewNowPlayingPage(
	highlightedTrackID string,
	contr *controller.Controller,
	conf *backend.NowPlayingPageConfig,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{contr: contr, conf: conf, sm: sm, pm: pm}}
	a.ExtendBaseWidget(a)
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
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(a.title, nil, nil, nil, a.tracklist))
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

func (a *NowPlayingPage) OnSongChange(song, lastScrobbledIfAny *mediaprovider.Track) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
}

func (a *NowPlayingPage) Reload() {
	a.load("")
}

func (a *NowPlayingPage) onPlayTrackAt(tracknum int) {
	_ = a.pm.PlayTrackAt(tracknum)
}

func (a *NowPlayingPage) onRemoveSelectedFromQueue() {
	a.pm.RemoveTracksFromQueue(a.tracklist.SelectedTrackIndexes())
	a.tracklist.UnselectAll()
	a.Reload()
}

// does not make calls to server - can safely be run in UI callbacks
func (a *NowPlayingPage) load(highlightedTrackID string) {
	queue := a.pm.GetPlayQueue()
	a.tracklist.Tracks = queue
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	if highlightedTrackID != "" {
		a.tracklist.SelectAndScrollToTrack(highlightedTrackID)
	}
}

func (s *nowPlayingPageState) Restore() Page {
	return NewNowPlayingPage("", s.contr, s.conf, s.sm, s.pm)
}
