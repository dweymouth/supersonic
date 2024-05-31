package browsing

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/dominantcolor"
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
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NowPlayingPage struct {
	widget.BaseWidget

	nowPlayingPageState

	// volatile state
	nowPlaying   mediaprovider.MediaItem
	nowPlayingID string
	curLyricsID  string // id of track currently shown in lyrics
	curRelatedID string // id of track currrently used to populate related list
	totalTime    float64
	lastPlayPos  float64
	queue        []*mediaprovider.Track
	related      []*mediaprovider.Track

	lyricLock   sync.Mutex
	relatedLock sync.Mutex

	// widgets for render
	background   *canvas.LinearGradient
	queueList    *widgets.PlayQueueList
	relatedList  *widgets.PlayQueueList
	lyricsViewer *widgets.LyricsViewer
	card         *widgets.LargeNowPlayingCard
	statusLabel  *widget.Label
	tabs         *container.AppTabs
	tabLoading   *widgets.LoadingDots
	container    *fyne.Container

	// cancel funcs for background fetch tasks
	lyricFetchCancel   context.CancelFunc
	imageLoadCancel    context.CancelFunc
	relatedFetchCancel context.CancelFunc
}

type nowPlayingPageState struct {
	conf     *backend.NowPlayingPageConfig
	contr    *controller.Controller
	pool     *util.WidgetPool
	sm       *backend.ServerManager
	pm       *backend.PlaybackManager
	im       *backend.ImageManager
	mp       mediaprovider.MediaProvider
	canRate  bool
	canShare bool
	lrcLib   bool
}

func NewNowPlayingPage(
	conf *backend.NowPlayingPageConfig,
	contr *controller.Controller,
	pool *util.WidgetPool,
	sm *backend.ServerManager,
	im *backend.ImageManager,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	canRate bool,
	canShare bool,
	lrcLibEnabled bool,
) *NowPlayingPage {
	state := nowPlayingPageState{
		conf: conf, contr: contr, pool: pool, sm: sm, im: im, pm: pm, mp: mp, canRate: canRate, canShare: canShare, lrcLib: lrcLibEnabled,
	}
	if page, ok := pool.Obtain(util.WidgetTypeNowPlayingPage).(*NowPlayingPage); ok && page != nil {
		page.nowPlayingPageState = state
		page.Reload()
		return page
	}
	a := &NowPlayingPage{nowPlayingPageState: state}
	a.ExtendBaseWidget(a)

	pm.OnPaused(a.formatStatusLine)
	pm.OnPlaying(a.formatStatusLine)
	pm.OnStopped(a.formatStatusLine)

	a.card = widgets.NewLargeNowPlayingCard()
	a.card.OnAlbumNameTapped = func() {
		contr.NavigateTo(controller.AlbumRoute(a.nowPlaying.Metadata().AlbumID))
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
	a.relatedList = widgets.NewPlayQueueList(a.im)
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
	a.relatedList.OnDownload = a.queueList.OnDownload
	a.relatedList.OnShare = a.queueList.OnShare
	a.relatedList.OnAddToPlaylist = a.queueList.OnAddToPlaylist
	a.relatedList.OnShowArtistPage = a.queueList.OnShowArtistPage
	a.relatedList.OnRemoveFromQueue = a.queueList.OnRemoveFromQueue
	a.relatedList.OnSetRating = a.queueList.OnSetRating
	a.relatedList.OnSetFavorite = a.queueList.OnSetFavorite
	a.relatedList.OnPlayTrackAt = func(idx int) {
		a.pm.LoadTracks(a.related, backend.Replace, false)
		a.pm.PlayTrackAt(idx)
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
		} else if a.conf.InitialView == "Related" {
			initialTab = 2
		}
		_ = initialTab
		paddedLayout := &layouts.PercentPadLayout{
			LeftRightObjectPercent: .8,
			TopBottomObjectPercent: .8,
		}
		a.tabs = container.NewAppTabs(
			container.NewTabItem("Play Queue",
				container.NewBorder(layout.NewSpacer(), nil, nil, nil, a.queueList)),
			container.NewTabItem("Lyrics", a.lyricsViewer),
			container.NewTabItem("Related", a.relatedList),
		)
		a.tabs.SelectIndex(initialTab)
		a.tabs.OnSelected = func(*container.TabItem) {
			idx := a.tabs.SelectedIndex()
			a.saveSelectedTab(idx)
			if idx == 1 /*lyrics*/ {
				a.updateLyrics()
			} else if idx == 2 /*related*/ {
				a.updateRelatedList()
			}
		}
		a.tabLoading = widgets.NewLoadingDots()
		if initialTab == 1 /*lyrics*/ {
			a.updateLyrics()
		} else if initialTab == 2 /*related*/ {
			a.updateRelatedList()
		}
		c := theme.Color(myTheme.ColorNamePageBackground)
		a.background = canvas.NewLinearGradient(c, c, 0)

		mainContent := container.NewGridWithColumns(2,
			container.New(paddedLayout, a.card),
			container.New(paddedLayout,
				container.NewStack(
					util.AddHeaderBackgroundWithColorName(
						a.tabs, myTheme.ColorNameNowPlayingPanel),
					container.NewCenter(a.tabLoading))))
		a.container = container.NewStack(
			a.background,
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

func (a *NowPlayingPage) OnSongChange(song mediaprovider.MediaItem, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlaying = song
	if a.imageLoadCancel != nil {
		a.imageLoadCancel()
	}
	a.nowPlayingID = sharedutil.MediaItemIDOrEmptyStr(song)
	a.queueList.SetNowPlaying(a.nowPlayingID)
	a.relatedList.SetNowPlaying(a.nowPlayingID)

	a.card.Update(song)
	if song == nil {
		a.card.SetCoverImage(nil)
	} else {
		a.imageLoadCancel = a.im.GetFullSizeCoverArtAsync(song.Metadata().CoverArtID, a.onImageLoaded)
	}

	if a.tabs != nil && a.tabs.SelectedIndex() == 1 /*lyrics*/ {
		a.updateLyrics()
	} else if a.tabs != nil && a.tabs.SelectedIndex() == 2 /*related*/ {
		a.updateRelatedList()
	}
}

func (a *NowPlayingPage) onImageLoaded(img image.Image, err error) {
	if err != nil {
		log.Printf("error loading cover art: %v\n", err)
		return
	}
	a.card.SetCoverImage(img)
	c := dominantcolor.Find(img)
	if c == a.background.StartColor {
		return
	}

	evenFrame := true
	anim := canvas.NewColorRGBAAnimation(
		a.background.StartColor, c, 75*time.Millisecond, func(c color.Color) {
			// reduce fps to reduce mem allocations
			if evenFrame {
				a.background.StartColor = c
				a.background.Refresh()
			}
			evenFrame = !evenFrame
		})
	anim.Start()

}

func (a *NowPlayingPage) updateLyrics() {
	a.lyricLock.Lock()
	defer a.lyricLock.Unlock()

	if a.lyricFetchCancel != nil {
		a.lyricFetchCancel()
	}

	if a.nowPlayingID == a.curLyricsID {
		if a.nowPlayingID != "" {
			// just need to sync the current time
			a.lyricsViewer.OnSeeked(a.lastPlayPos)
			return
		}
	}
	if a.nowPlaying == nil || a.nowPlaying.Metadata().Type == mediaprovider.MediaItemTypeRadioStation {
		a.lyricsViewer.SetLyrics(nil)
		a.curLyricsID = ""
		return
	}
	a.curLyricsID = a.nowPlayingID
	ctx, cancel := context.WithCancel(context.Background())
	a.lyricFetchCancel = cancel
	a.tabLoading.Start()
	// set the widget to an empty (not nil) lyric during fetch
	// to keep it from showing "Lyrics not available"
	a.lyricsViewer.SetLyrics(&mediaprovider.Lyrics{Synced: true,
		Lines: []mediaprovider.LyricLine{{Text: ""}}})
	tr, _ := a.nowPlaying.(*mediaprovider.Track)
	go a.fetchLyrics(ctx, tr)
}

func (a *NowPlayingPage) fetchLyrics(ctx context.Context, song *mediaprovider.Track) {
	var lyrics *mediaprovider.Lyrics
	var err error
	if lp, ok := a.sm.Server.(mediaprovider.LyricsProvider); ok {
		if lyrics, err = lp.GetLyrics(song); err != nil {
			log.Printf("Error fetching lyrics: %v", err)
		}
	}
	if lyrics == nil {
		lyrics, err = backend.FetchLrcLibLyrics(song.Title, song.ArtistNames[0], song.Album, song.Duration)
		if err != nil {
			log.Println(err.Error())
		}
	}
	select {
	case <-ctx.Done():
		return
	default:
		a.lyricLock.Lock()
		a.tabLoading.Stop()
		a.lyricsViewer.SetLyrics(lyrics)
		if lyrics != nil {
			a.lyricsViewer.OnSeeked(a.lastPlayPos)
		}
		a.lyricLock.Unlock()
	}
}

func (a *NowPlayingPage) updateRelatedList() {
	a.relatedLock.Lock()
	defer a.relatedLock.Unlock()

	if a.relatedFetchCancel != nil {
		a.relatedFetchCancel()
	}
	if a.curRelatedID == a.nowPlayingID {
		return
	}
	a.curRelatedID = a.nowPlayingID
	if a.nowPlayingID == "" {
		a.relatedList.SetTracks(nil)
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.relatedFetchCancel = cancel
	a.tabLoading.Start()
	a.relatedList.SetTracks(nil)
	go func(ctx context.Context) {
		related, err := a.mp.GetSongRadio(a.nowPlayingID, 30)
		if err != nil {
			log.Printf("failed to get similar tracks: %s", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		default:
			a.relatedLock.Lock()
			a.related = related
			a.relatedList.SetTracks(a.related)
			a.tabLoading.Stop()
			a.relatedLock.Unlock()
		}
	}(ctx)
}

func (a *NowPlayingPage) OnPlayQueueChange() {
	a.Reload()
}

func (a *NowPlayingPage) Reload() {
	a.card.DisableRating = !a.canRate
	a.queueList.DisableRating = !a.canRate
	a.queueList.DisableSharing = !a.canShare
	a.relatedList.DisableRating = !a.canRate
	a.relatedList.DisableSharing = !a.canShare

	// TODO
	//a.queue = a.pm.GetPlayQueue()
	a.queueList.SetTracks(a.queue)
	a.totalTime = 0.0
	for _, tr := range a.queue {
		a.totalTime += float64(tr.Duration)
	}
	a.formatStatusLine()
}

func (s *nowPlayingPageState) Restore() Page {
	return NewNowPlayingPage(s.conf, s.contr, s.pool, s.sm, s.im, s.pm, s.mp, s.canRate, s.canShare, s.lrcLib)
}

var _ CanShowPlayTime = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) OnPlayTimeUpdate(curTime, _ float64, seeked bool) {
	a.lastPlayPos = curTime
	a.formatStatusLine()
	if a.tabs == nil || a.tabs.SelectedIndex() != 1 /*lyrics*/ {
		return
	}
	a.lyricLock.Lock()
	defer a.lyricLock.Unlock()
	if seeked {
		a.lyricsViewer.OnSeeked(curTime)
	} else {
		a.lyricsViewer.UpdatePlayPos(curTime)
	}
}

var _ CanSelectAll = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) SelectAll() {
	a.queueList.SelectAll()
}

var _ fyne.Tappable = (*NowPlayingPage)(nil)

func (a *NowPlayingPage) Tapped(*fyne.PointEvent) {
	a.queueList.UnselectAll()
}

func (a *NowPlayingPage) Refresh() {
	if a.background != nil {
		c := theme.Color(myTheme.ColorNamePageBackground)
		if c != a.background.EndColor {
			a.background.EndColor = c
			a.background.Refresh()
		}
	}
	a.BaseWidget.Refresh()
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
	case 2:
		tabName = "Related"
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
		dur = float64(np.Metadata().Duration)
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
