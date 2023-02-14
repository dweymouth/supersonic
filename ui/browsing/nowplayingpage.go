package browsing

import (
	"supersonic/backend"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
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
	contr controller.Controller
	sm    *backend.ServerManager
	pm    *backend.PlaybackManager
	nav   func(Route)
}

func NewNowPlayingPage(
	contr controller.Controller,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	nav func(Route),
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{contr: contr, sm: sm, pm: pm, nav: nav}}
	a.ExtendBaseWidget(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns([]widgets.TracklistColumn{
		widgets.ColumnArtist, widgets.ColumnAlbum, widgets.ColumnTime})
	a.tracklist.AutoNumber = true
	a.tracklist.DisablePlaybackMenu = true
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.tracklist.OnAddToPlaylist = a.contr.DoAddTracksToPlaylistWorkflow
	a.tracklist.AuxiliaryMenuItems = []*fyne.MenuItem{
		fyne.NewMenuItem("Remove from queue", a.onRemoveSelectedFromQueue),
	}
	a.title = widget.NewRichTextWithText("Now Playing")
	a.title.Segments[0].(*widget.TextSegment).Style.SizeName = widget.RichTextStyleHeading.SizeName
	a.container = container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 5, PadBottom: 15},
		container.NewBorder(a.title, nil, nil, nil, a.tracklist))
	a.loadAsync()
	return a
}

func (a *NowPlayingPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *NowPlayingPage) Save() SavedPage {
	nps := a.nowPlayingPageState
	return &nps
}

func (a *NowPlayingPage) Route() Route {
	return NowPlayingRoute()
}

func (a *NowPlayingPage) Tapped(*fyne.PointEvent) {
	a.tracklist.UnselectAll()
}

func (a *NowPlayingPage) SelectAll() {
	a.tracklist.SelectAll()
}

func (a *NowPlayingPage) OnSongChange(song *subsonic.Child, lastScrobbledIfAny *subsonic.Child) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(lastScrobbledIfAny)
}

func (a *NowPlayingPage) Reload() {
	a.loadAsync()
}

func (a *NowPlayingPage) onPlayTrackAt(tracknum int) {
	_ = a.pm.PlayTrackAt(tracknum)
}

func (a *NowPlayingPage) onRemoveSelectedFromQueue() {
	a.pm.RemoveTracksFromQueue(a.tracklist.SelectedTrackIndexes())
	a.tracklist.UnselectAll()
	go a.Reload()
}

func (a *NowPlayingPage) loadAsync() {
	go func() {
		queue := a.pm.GetPlayQueue()
		a.tracklist.Tracks = queue
		a.tracklist.SetNowPlaying(a.nowPlayingID)
	}()
}

func (s *nowPlayingPageState) Restore() Page {
	return NewNowPlayingPage(s.contr, s.sm, s.pm, s.nav)
}
