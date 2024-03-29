package browsing

import (
	"context"
	"fmt"
	"image"
	"log"
	"slices"
	"strings"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NowPlayingPage struct {
	widget.BaseWidget

	nowPlayingPageState

	queue           []*mediaprovider.Track
	queueList       *widgets.PlayQueueList
	lyricsViewer    *widgets.LyricsViewer
	imageLoadCancel context.CancelFunc
	card            *widgets.LargeNowPlayingCard
	statusLabel     *widget.Label
	totalTime       float64
	nowPlayingID    string
	albumID         string
	container       *fyne.Container
}

type nowPlayingPageState struct {
	conf     *backend.NowPlayingPageConfig
	contr    *controller.Controller
	pool     *util.WidgetPool
	sm       *backend.ServerManager
	pm       *backend.PlaybackManager
	im       *backend.ImageManager
	canRate  bool
	canShare bool
}

func NewNowPlayingPage(
	conf *backend.NowPlayingPageConfig,
	contr *controller.Controller,
	pool *util.WidgetPool,
	sm *backend.ServerManager,
	im *backend.ImageManager,
	pm *backend.PlaybackManager,
	canRate bool,
	canShare bool,
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{
		conf: conf, contr: contr, pool: pool, sm: sm, im: im, pm: pm, canRate: canRate, canShare: canShare,
	}}
	a.ExtendBaseWidget(a)

	pm.OnPaused(a.formatStatusLine)
	pm.OnPlaying(a.formatStatusLine)
	pm.OnStopped(a.formatStatusLine)

	a.card = widgets.NewLargeNowPlayingCard()
	a.card.DisableRating = !canRate
	a.card.OnAlbumNameTapped = func() {
		contr.NavigateTo(controller.AlbumRoute(a.albumID))
	}
	a.card.OnArtistNameTapped = func(artistID string) {
		contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	a.card.OnSetFavorite = func(fav bool) {
		a.contr.SetTrackFavorites([]string{a.nowPlayingID}, fav)
	}
	a.card.OnSetRating = func(rating int) {
		a.contr.SetTrackRatings([]string{a.nowPlayingID}, rating)
	}

	a.queueList = widgets.NewPlayQueueList(a.im)
	a.queueList.DisableRating = !canRate
	a.queueList.DisableSharing = !canShare
	a.queueList.OnReorderTracks = a.doSetNewTrackOrder
	a.queueList.OnDownload = contr.ShowDownloadDialog
	a.queueList.OnShare = func(tracks []*mediaprovider.Track) {
		if len(tracks) > 0 {
			a.contr.ShowShareDialog(tracks[0].ID)
		}
	}
	a.queueList.OnAddToPlaylist = contr.DoAddTracksToPlaylistWorkflow
	a.queueList.OnPlayTrackAt = func(tracknum int) {
		_ = a.pm.PlayTrackAt(tracknum)
	}
	a.queueList.OnShowArtistPage = func(artistID string) {
		a.contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	a.queueList.OnRemoveFromQueue = func(trackIDs []string) {
		a.queueList.UnselectAll()
		a.pm.RemoveTracksFromQueue(trackIDs)
	}
	a.queueList.OnSetRating = func(trackIDs []string, rating int) {
		contr.SetTrackRatings(trackIDs, rating)
		if slices.Contains(trackIDs, a.nowPlayingID) {
			a.card.SetDisplayedRating(rating)
		}
	}
	a.queueList.OnSetFavorite = func(trackIDs []string, fav bool) {
		contr.SetTrackFavorites(trackIDs, fav)
		if slices.Contains(trackIDs, a.nowPlayingID) {
			a.card.SetDisplayedFavorite(fav)
		}
	}

	a.lyricsViewer = widgets.NewLyricsViewer()
	a.statusLabel = widget.NewLabel("Stopped")

	a.Reload()
	return a
}

func (a *NowPlayingPage) CreateRenderer() fyne.WidgetRenderer {
	if a.container == nil {
		initialTab := 0
		if a.conf.InitialView == "Lyrics" {
			initialTab = 1
		}
		_ = initialTab
		paddedLayout := &layouts.PercentPadLayout{
			LeftRightObjectPercent: .8,
			TopBottomObjectPercent: .8,
		}
		tabs := container.NewAppTabs(
			container.NewTabItem("Play Queue",
				container.NewBorder(layout.NewSpacer(), nil, nil, nil, a.queueList)),
			container.NewTabItem("Lyrics", a.lyricsViewer),
		)
		tabs.SelectIndex(initialTab)
		tabs.OnSelected = func(*container.TabItem) {
			a.saveSelectedTab(tabs.SelectedIndex())
		}
		mainContent := container.NewGridWithColumns(2,
			container.New(paddedLayout, a.card),
			container.New(paddedLayout,
				util.AddHeaderBackground(tabs)))
		a.container = container.NewStack(
			mainContent,
			container.NewVBox(
				layout.NewSpacer(),
				container.NewBorder(nil, nil, util.NewHSpace(1), util.NewHSpace(1),
					myTheme.NewThemedRectangle(theme.ColorNameInputBorder)),
				a.statusLabel,
			),
		)
	}
	return widget.NewSimpleRenderer(a.container)
}

func (a *NowPlayingPage) Save() SavedPage {
	if a.imageLoadCancel != nil {
		a.imageLoadCancel()
	}
	nps := a.nowPlayingPageState
	a.pool.Release(util.WidgetTypeNowPlayingPage, a)
	return &nps
}

func (a *NowPlayingPage) Route() controller.Route {
	return controller.NowPlayingRoute("")
}

var _ CanShowNowPlaying = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnSongChange(song, lastScrobbledIfAny *mediaprovider.Track) {
	if a.imageLoadCancel != nil {
		a.imageLoadCancel()
	}
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(song)
	a.queueList.SetNowPlaying(a.nowPlayingID)

	a.albumID = sharedutil.AlbumIDOrEmptyStr(song)
	a.card.Update(song)
	if song == nil {
		a.card.SetCoverImage(nil)
		return
	}

	a.imageLoadCancel = a.im.GetFullSizeCoverArtAsync(song.CoverArtID, func(img image.Image, err error) {
		if err != nil {
			log.Printf("error loading cover art: %v\n", err)
		} else {
			a.card.SetCoverImage(img)
		}
	})

	go func() {
		if lp, ok := a.sm.Server.(mediaprovider.LyricsProvider); ok {
			lyrics, err := lp.GetLyrics(song)
			if err == nil {
				a.lyricsViewer.SetLyrics(lyrics)
			} else {
				log.Printf("Error fetching lyrics: %v", err)
				a.lyricsViewer.SetLyrics(nil)
			}
		}
	}()
}

func (a *NowPlayingPage) OnPlayQueueChange() {
	a.Reload()
}

func (a *NowPlayingPage) Reload() {
	a.queue = a.pm.GetPlayQueue()
	a.queueList.SetTracks(a.queue)
	a.totalTime = 0.0
	for _, tr := range a.queue {
		a.totalTime += float64(tr.Duration)
	}
	a.formatStatusLine()
}

func (s *nowPlayingPageState) Restore() Page {
	if page := s.pool.Obtain(util.WidgetTypeNowPlayingPage).(*NowPlayingPage); page != nil {
		page.Reload()
		return page
	}
	return NewNowPlayingPage(s.conf, s.contr, s.pool, s.sm, s.im, s.pm, s.canRate, s.canShare)
}

var _ CanShowPlayTime = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnPlayTimeUpdate(_, _ float64) {
	a.formatStatusLine()
}

var _ CanSelectAll = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) SelectAll() {
	a.queueList.SelectAll()
}

var _ fyne.Tappable = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) Tapped(*fyne.PointEvent) {
	a.queueList.UnselectAll()
}

func (a *NowPlayingPage) doSetNewTrackOrder(trackIDs []string, op sharedutil.TrackReorderOp) {
	trackIDSet := sharedutil.ToSet(trackIDs)
	idxs := make([]int, 0, len(trackIDs))
	for i, tr := range a.queue {
		if _, ok := trackIDSet[tr.ID]; ok {
			idxs = append(idxs, i)
		}
	}
	newTracks := sharedutil.ReorderTracks(a.queue, idxs, op)
	a.pm.UpdatePlayQueue(newTracks)
}

func (a *NowPlayingPage) saveSelectedTab(tabNum int) {
	var tabName string
	switch tabNum {
	case 0:
		tabName = "Play Queue"
	case 1:
		tabName = "Lyrics"
	}
	a.conf.InitialView = tabName
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
