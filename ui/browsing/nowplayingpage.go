package browsing

import (
	"context"
	"fmt"
	"image"
	"log"
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
	imageLoadCancel context.CancelFunc
	card            *widgets.LargeNowPlayingCard
	statusLabel     *widget.Label
	totalTime       float64
	nowPlayingID    string
	albumID         string
	container       *fyne.Container
}

type nowPlayingPageState struct {
	contr   *controller.Controller
	pool    *util.WidgetPool
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	canRate bool
}

func NewNowPlayingPage(
	contr *controller.Controller,
	pool *util.WidgetPool,
	im *backend.ImageManager,
	pm *backend.PlaybackManager,
	canRate bool,
) *NowPlayingPage {
	a := &NowPlayingPage{nowPlayingPageState: nowPlayingPageState{
		contr: contr, pool: pool, im: im, pm: pm, canRate: canRate,
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
	a.queueList.OnReorderTracks = a.doSetNewTrackOrder
	a.queueList.OnDownload = contr.ShowDownloadDialog
	a.queueList.OnSetRating = contr.SetTrackRatings
	a.queueList.OnSetFavorite = contr.SetTrackFavorites
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

	a.statusLabel = widget.NewLabel("Stopped")

	a.Reload()
	return a
}

func (a *NowPlayingPage) CreateRenderer() fyne.WidgetRenderer {
	if a.container == nil {
		paddedLayout := &layouts.PercentPadLayout{
			LeftRightObjectPercent: .8,
			TopBottomObjectPercent: .8,
		}
		mainContent := container.NewGridWithColumns(2,
			container.New(paddedLayout, a.card),
			container.New(paddedLayout,
				util.AddHeaderBackground(
					container.NewAppTabs(
						container.NewTabItem("Play Queue",
							container.NewBorder(layout.NewSpacer(), nil, nil, nil, a.queueList)),
						container.NewTabItem("Lyrics", layout.NewSpacer()),
					))))
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
	return NewNowPlayingPage(s.contr, s.pool, s.im, s.pm, s.canRate)
}

var _ CanShowPlayTime = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnPlayTimeUpdate(_, _ float64) {
	a.formatStatusLine()
}

func (a *NowPlayingPage) doSetNewTrackOrder(trackIDs []string, op sharedutil.TrackReorderOp) {
	idxs := make([]int, 0, len(trackIDs))
	for i, tr := range a.queue {
		if sharedutil.SliceContains(trackIDs, tr.ID) {
			idxs = append(idxs, i)
		}
	}
	newTracks := sharedutil.ReorderTracks(a.queue, idxs, op)
	a.pm.UpdatePlayQueue(newTracks)
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
