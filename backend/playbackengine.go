package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/dweymouth/supersonic/sharedutil"
)

// TODO: make thread-safe

var (
	ReplayGainNone  = player.ReplayGainNone.String()
	ReplayGainAlbum = player.ReplayGainAlbum.String()
	ReplayGainTrack = player.ReplayGainTrack.String()
	ReplayGainAuto  = "Auto"
)

type InsertQueueMode int

const (
	Replace InsertQueueMode = iota
	InsertNext
	Append
)

// The playback loop mode (LoopNone, LoopAll, LoopOne).
type LoopMode int

const (
	LoopNone LoopMode = iota
	LoopAll
	LoopOne
)

type PlaybackState = player.State

type PlaybackStatus struct {
	State    PlaybackState
	TimePos  float64
	Duration float64
}

type playbackEngine struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	sm            *ServerManager
	audiocache    *AudioCache
	player        player.BasePlayer

	playTimeStopwatch   util.Stopwatch
	curTrackDuration    float64
	latestTrackPosition float64 // cleared by checkScrobble
	callbacksDisabled   bool

	playQueue     []mediaprovider.MediaItem
	nowPlayingIdx int
	isRadio       bool
	loopMode      LoopMode

	pauseAfterCurrent bool // flag to pause playback after current track ends

	// flags for handleOnTrackChange / handleOnStopped callbacks - reset to false in the callbacks
	wasStopped       bool // true iff player was stopped before handleOnTrackChange invocation
	alreadyScrobbled bool // true iff the previously-playing track was already scrobbled

	// if >= 0, track number that was requested by PlayTrackAt
	// onTrackChange callback should set nowPlayingIdx to this,
	// and reset this to -1
	pendingTrackChangeNum int

	pendingPlayerChange       bool
	pendingPlayerChangeStatus player.Status

	// Whether we need to set the next track on the Player
	// before the current track completes (normally when 10 seconds remain
	// in the time pos polling function)
	needToSetNextTrack bool

	// to pass to onSongChange listeners; clear once listeners have been called
	lastScrobbled *mediaprovider.Track
	playbackCfg   *PlaybackConfig
	scrobbleCfg   *ScrobbleConfig
	transcodeCfg  *TranscodingConfig
	replayGainCfg ReplayGainConfig

	// registered callbacks
	onBeforeSongChange []func(next mediaprovider.MediaItem)
	onSongChange       []func(nowPlaying mediaprovider.MediaItem, justScrobbledIfAny *mediaprovider.Track)
	onPlayTimeUpdate   []func(float64, float64, bool)
	onLoopModeChange   []func(LoopMode)
	onVolumeChange     []func(int)
	onAudioInfoChange  []func()
	onSeek             []func()
	onPaused           []func()
	onStopped          []func()
	onPlaying          []func()
	onQueueChange      []func()
}

func NewPlaybackEngine(
	ctx context.Context,
	s *ServerManager,
	c *AudioCache,
	p player.BasePlayer,
	playbackCfg *PlaybackConfig,
	scrobbleCfg *ScrobbleConfig,
	transcodeCfg *TranscodingConfig,
) *playbackEngine {
	// clamp to 99% to avoid any possible rounding issues
	scrobbleCfg.ThresholdPercent = clamp(scrobbleCfg.ThresholdPercent, 0, 99)
	pm := &playbackEngine{
		ctx:           ctx,
		sm:            s,
		audiocache:    c,
		player:        p,
		playbackCfg:   playbackCfg,
		scrobbleCfg:   scrobbleCfg,
		transcodeCfg:  transcodeCfg,
		nowPlayingIdx: -1,
		wasStopped:    true,
	}
	switch playbackCfg.RepeatMode {
	case "All":
		pm.loopMode = LoopAll
	case "One":
		pm.loopMode = LoopOne
	}

	pm.registerPlayerCallbacks(p)
	s.OnLogout(func() {
		pm.StopAndClearPlayQueue()
	})

	return pm
}

func (p *playbackEngine) registerPlayerCallbacks(pl player.BasePlayer) {
	pl.OnTrackChange(p.handleOnTrackChange)
	pl.OnSeek(func() {
		p.handleTimePosUpdate(true)
		p.invokeNoArgCallbacks(p.onSeek)
	})
	pl.OnStopped(p.handleOnStopped)
	pl.OnPaused(func() {
		p.playTimeStopwatch.Stop()
		p.stopPollTimePos()
		p.invokeNoArgCallbacks(p.onPaused)
	})
	pl.OnPlaying(func() {
		p.playTimeStopwatch.Start()
		p.startPollTimePos()
		p.invokeNoArgCallbacks(p.onPlaying)
	})
	pl.OnVolumeChange(func(vol int) {
		for _, cb := range p.onVolumeChange {
			cb(vol)
		}
	})
	pl.OnAudioInfoChange(func() {
		for _, cb := range p.onAudioInfoChange {
			cb()
		}
	})
}

func (p *playbackEngine) unregisterPlayerCallbacks(pl player.BasePlayer) {
	pl.OnPaused(nil)
	pl.OnPlaying(nil)
	pl.OnStopped(nil)
	pl.OnSeek(nil)
	pl.OnTrackChange(nil)
	pl.OnVolumeChange(nil)
	pl.OnAudioInfoChange(nil)
}

func (p *playbackEngine) SetPlayer(pl player.BasePlayer) error {
	needToUnpause := false

	// Get fresh status from current player (ignore any cached pending state)
	stat := p.CurrentPlayer().GetStatus()

	// Always stop polling when switching players to prevent stale goroutines
	p.stopPollTimePos()

	// Clear any stale pending state before evaluating current state
	p.pendingPlayerChange = false

	// Only cache playback state if actively playing - we'll resume on the new player
	if stat.State == player.Playing {
		needToUnpause = true
		p.pendingPlayerChangeStatus = stat
		p.pendingPlayerChange = true
	}
	// For Stopped or Paused states, don't cache - the old player's position
	// isn't meaningful for the new player
	p.unregisterPlayerCallbacks(p.player)
	if err := p.player.Stop(true); err != nil {
		log.Printf("failed to stop player: %v", err)
	}

	oldVol := p.player.GetVolume()
	if _, isMPV := p.player.(*mpv.Player); !isMPV {
		p.player.Destroy()
	}
	p.player = pl
	p.registerPlayerCallbacks(pl)

	if needToUnpause {
		// Clear pendingPlayerChange BEFORE playTrackAt, because playTrackAt
		// triggers OnPlaying which starts time polling. If we clear it after,
		// the polling goroutine may see stale cached status.
		startTime := p.pendingPlayerChangeStatus.TimePos
		p.pendingPlayerChange = false
		p.playTrackAt(p.nowPlayingIdx, startTime)
	}
	vol := pl.GetVolume()
	if oldVol != vol {
		for _, cb := range p.onVolumeChange {
			cb(vol)
		}
	}
	return nil
}

func (p *playbackEngine) PlayTrackAt(idx int) error {
	return p.playTrackAt(idx, 0)
}

func (p *playbackEngine) playTrackAt(idx int, startTime float64) error {
	if l := len(p.playQueue); idx < 0 || idx >= l {
		return fmt.Errorf("track index (%d) out of range (0-%d)", idx, l)
	}
	// scrobble current track if needed
	p.checkScrobble()
	p.alreadyScrobbled = true
	p.pendingTrackChangeNum = idx
	err := p.setTrack(idx, false, startTime)
	return err
}

// Gets the curently playing media item, if any.
func (p *playbackEngine) NowPlaying() mediaprovider.MediaItem {
	idx := p.nowPlayingIdx
	queue := p.playQueue
	if idx < 0 || len(queue) == 0 || idx >= len(queue) || p.player.GetStatus().State == player.Stopped {
		return nil
	}
	return queue[idx]
}

func (p *playbackEngine) NowPlayingIndex() int {
	return int(p.nowPlayingIdx)
}

// SetNowPlayingIndex sets the now playing index without triggering playback.
// This is used to sync with external jukebox players (like MPD) that are already playing.
func (p *playbackEngine) SetNowPlayingIndex(idx int) {
	if idx >= 0 && idx < len(p.playQueue) {
		p.nowPlayingIdx = idx
		// Trigger song change callback so UI updates
		nowPlaying := p.playQueue[p.nowPlayingIdx]
		for _, cb := range p.onSongChange {
			cb(nowPlaying, nil)
		}
	}
}

// SyncQueueFromExternal syncs the local queue with an external jukebox player's queue
// without affecting the remote player's state. This is used when connecting to MPD
// to show what's currently playing without interrupting playback.
func (p *playbackEngine) SyncQueueFromExternal(tracks []*mediaprovider.Track, currentIdx int) {
	// Clear loaded queue state if player supports it
	if qp, ok := p.player.(player.QueuePlayer); ok {
		qp.ClearLoadedQueue()
	}

	// Set the play queue without sending commands to the remote player
	p.playQueue = make([]mediaprovider.MediaItem, len(tracks))
	for i, tr := range tracks {
		p.playQueue[i] = tr
	}

	// Set the now playing index
	if currentIdx >= 0 && currentIdx < len(tracks) {
		p.nowPlayingIdx = currentIdx
		// Trigger song change callback so UI updates
		nowPlaying := p.playQueue[p.nowPlayingIdx]
		for _, cb := range p.onSongChange {
			cb(nowPlaying, nil)
		}
	} else {
		p.nowPlayingIdx = -1
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
}

func (p *playbackEngine) SetLoopMode(loopMode LoopMode) {
	p.loopMode = loopMode
	if p.nowPlayingIdx >= 0 {
		// TODO - don't need when going from LoopNone to LoopAll
		// if not on last track
		p.handleNextTrackUpdated()
	}

	for _, cb := range p.onLoopModeChange {
		cb(loopMode)
	}
}

func (p *playbackEngine) GetLoopMode() LoopMode {
	return p.loopMode
}

func (p *playbackEngine) PlaybackStatus() PlaybackStatus {
	stat := p.pendingPlayerChangeStatus
	if !p.pendingPlayerChange {
		stat = p.CurrentPlayer().GetStatus()
	}
	return PlaybackStatus{
		State:    stat.State,
		TimePos:  stat.TimePos,
		Duration: p.curTrackDuration,
	}
}

func (p *playbackEngine) SetVolume(vol int) error {
	vol = clamp(vol, 0, 100)
	if err := p.player.SetVolume(vol); err != nil {
		return err
	}
	for _, cb := range p.onVolumeChange {
		cb(vol)
	}
	return nil
}

func (p *playbackEngine) CurrentPlayer() player.BasePlayer {
	return p.player
}

func (p *playbackEngine) SeekNext() error {
	if p.PlaybackStatus().State == player.Stopped {
		return nil
	}
	return p.PlayTrackAt(p.nowPlayingIdx + 1)
}

func (p *playbackEngine) SeekBackOrPrevious() error {
	if p.nowPlayingIdx == 0 || p.PlaybackStatus().TimePos > 3 {
		return p.player.SeekSeconds(0)
	}
	return p.PlayTrackAt(p.nowPlayingIdx - 1)
}

func (p *playbackEngine) SeekFwdBackN(n int) error {
	idx := p.nowPlayingIdx
	if n < 0 && p.PlaybackStatus().TimePos > 3 {
		n += 1 // first seek back is just seek to beginning of current
	}
	if n == 0 || (idx == 0 && n < 0) {
		return p.player.SeekSeconds(0) // seek back in current song
	}

	lastIdx := len(p.playQueue) - 1
	newIdx := min(lastIdx, max(0, idx+n))

	if idx == lastIdx && n > 0 {
		newIdx = 0
	}
	return p.PlayTrackAt(newIdx)
}

// Seek to given absolute position in the current track by seconds.
func (p *playbackEngine) SeekSeconds(sec float64) error {
	if p.isRadio {
		return nil // can't seek radio stations
	}
	return p.player.SeekSeconds(sec)
}

func (p *playbackEngine) IsSeeking() bool {
	return p.player.IsSeeking()
}

func (p *playbackEngine) Stop() error {
	return p.player.Stop(false)
}

func (p *playbackEngine) SetPauseAfterCurrent(pauseAfterCurrent bool) {
	p.pauseAfterCurrent = pauseAfterCurrent
}

func (p *playbackEngine) Pause() error {
	return p.player.Pause()
}

func (p *playbackEngine) Continue() error {
	if p.pendingPlayerChange {
		p.pendingPlayerChange = false
		return p.playTrackAt(p.nowPlayingIdx, p.pendingPlayerChangeStatus.TimePos)
	}

	if p.PlaybackStatus().State == player.Stopped {
		return p.PlayTrackAt(0)
	}
	return p.player.Continue()
}

// Load items into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *playbackEngine) LoadItems(items []mediaprovider.MediaItem, insertQueueMode InsertQueueMode, shuffle bool) error {
	newItems := deepCopyMediaItemSlice(items)
	return p.doLoaditems(newItems, insertQueueMode, shuffle)
}

// Load tracks into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *playbackEngine) LoadTracks(tracks []*mediaprovider.Track, insertQueueMode InsertQueueMode, shuffle bool) error {
	newTracks := copyTrackSliceToMediaItemSlice(tracks)
	return p.doLoaditems(newTracks, insertQueueMode, shuffle)
}

func (p *playbackEngine) doLoaditems(items []mediaprovider.MediaItem, insertQueueMode InsertQueueMode, shuffle bool) error {
	// Clear loaded queue state if player supports it and we're modifying the queue
	if qp, ok := p.player.(player.QueuePlayer); ok {
		qp.ClearLoadedQueue()
	}

	if insertQueueMode == Replace {
		p.player.Stop(false)
		p.nowPlayingIdx = -1
		p.playQueue = nil
	}
	if nextChanged := len(items) > 0 && (insertQueueMode != Append || (p.nowPlayingIdx == len(p.playQueue)-1)); nextChanged {
		defer p.handleNextTrackUpdated()
	}

	if shuffle {
		rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	}

	insertIdx := len(p.playQueue)
	if insertQueueMode == InsertNext {
		insertIdx = p.nowPlayingIdx + 1
	}
	p.playQueue = append(p.playQueue[:insertIdx], append(items, p.playQueue[insertIdx:]...)...)

	// If replacing the queue and player supports bulk queue loading, load all tracks at once
	if insertQueueMode == Replace {
		if qp, ok := p.player.(player.QueuePlayer); ok {
			// Extract tracks from items (skip non-track items like radio stations)
			tracks := make([]*mediaprovider.Track, 0, len(items))
			for _, item := range items {
				if tr, ok := item.(*mediaprovider.Track); ok {
					tracks = append(tracks, tr)
				}
			}
			if len(tracks) > 0 && len(tracks) == len(items) {
				// All items are tracks, load them to the player's queue
				if err := qp.LoadQueue(tracks); err != nil {
					log.Printf("Failed to load queue to player: %v", err)
					// Fall back to track-by-track playback
				}
			}
		}
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
	return nil
}

func (p *playbackEngine) LoadRadioStation(radio *mediaprovider.RadioStation, insertMode InsertQueueMode) {
	// Radio stations can't be in a bulk-loaded queue
	if qp, ok := p.player.(player.QueuePlayer); ok {
		qp.ClearLoadedQueue()
	}

	if insertMode == Replace {
		p.player.Stop(false)
		p.nowPlayingIdx = -1
		p.playQueue = nil
	}
	if nextChanged := insertMode == InsertNext || (insertMode == Append && p.nowPlayingIdx == len(p.playQueue)-1); nextChanged {
		p.handleNextTrackUpdated()
	}
	insertIdx := len(p.playQueue)
	if insertMode == InsertNext {
		insertIdx = p.nowPlayingIdx + 1
	}
	new := make([]mediaprovider.MediaItem, len(p.playQueue)+1)
	firstHalf := p.playQueue[:insertIdx]
	copy(new, firstHalf)
	new[len(firstHalf)] = radio
	copy(new[len(firstHalf)+1:], p.playQueue[insertIdx:])
	p.playQueue = new

	p.invokeNoArgCallbacks(p.onQueueChange)
}

// Stop playback and clear the play queue.
// Uses force=true to avoid clearing remote server queues (e.g., MPD).
func (p *playbackEngine) StopAndClearPlayQueue() {
	changed := len(p.playQueue) > 0
	p.player.Stop(true)
	p.playQueue = nil
	p.nowPlayingIdx = -1
	if changed {
		p.invokeNoArgCallbacks(p.onQueueChange)
	}
}

func (p *playbackEngine) GetPlayQueue() []mediaprovider.MediaItem {
	return deepCopyMediaItemSlice(p.playQueue)
}

// Any time the user changes the favorite status of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *playbackEngine) OnTrackFavoriteStatusChanged(id string, fav bool) {
	if item := sharedutil.FindMediaItemByID(id, p.playQueue); item != nil {
		if tr, ok := item.(*mediaprovider.Track); ok {
			tr.Favorite = fav
		}
	}
}

// Any time the user changes the rating of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *playbackEngine) OnTrackRatingChanged(id string, rating int) {
	if item := sharedutil.FindMediaItemByID(id, p.playQueue); item != nil {
		if tr, ok := item.(*mediaprovider.Track); ok {
			tr.Rating = rating
		}
	}
}

// Replaces the play queue with the given set of tracks.
// Does not stop playback if the currently playing track is in the new queue,
// but updates the now playing index to point to the first instance of the track in the new queue.
func (p *playbackEngine) UpdatePlayQueue(items []mediaprovider.MediaItem) error {
	// Clear loaded queue state since we're modifying the queue
	if qp, ok := p.player.(player.QueuePlayer); ok {
		qp.ClearLoadedQueue()
	}

	newQueue := deepCopyMediaItemSlice(items)
	newNowPlayingIdx := -1
	if p.nowPlayingIdx >= 0 {
		nowPlayingID := p.playQueue[p.nowPlayingIdx].Metadata().ID
		for i, tr := range newQueue {
			if tr.Metadata().ID == nowPlayingID {
				newNowPlayingIdx = i
				break
			}
		}
	}

	p.playQueue = newQueue
	if p.nowPlayingIdx >= 0 && newNowPlayingIdx == -1 {
		return p.Stop()
	}
	if p.nowPlayingIdx >= 0 {
		p.handleNextTrackUpdated()
	}
	p.nowPlayingIdx = newNowPlayingIdx

	p.invokeNoArgCallbacks(p.onQueueChange)
	return nil
}

func (p *playbackEngine) RemoveTracksFromQueue(idxs []int) {
	// Clear loaded queue state since we're modifying the queue
	if qp, ok := p.player.(player.QueuePlayer); ok {
		qp.ClearLoadedQueue()
	}

	newQueue := make([]mediaprovider.MediaItem, 0, len(p.playQueue)-len(idxs))
	idxSet := sharedutil.ToSet(idxs)
	isPlayingTrackRemoved := false
	isNextPlayingTrackremoved := false
	nowPlaying := p.NowPlayingIndex()
	newNowPlaying := nowPlaying
	for i, tr := range p.playQueue {
		if _, ok := idxSet[i]; ok {
			if i < nowPlaying {
				// if removing a track earlier than the currently playing one (if any),
				// decrement new now playing index by one to account for new position in queue
				newNowPlaying--
			} else if i == nowPlaying {
				isPlayingTrackRemoved = true
				// If we are removing the currently playing track, we need to scrobble it
				p.checkScrobble()
				p.alreadyScrobbled = true
			} else if nowPlaying >= 0 && i == nowPlaying+1 {
				isNextPlayingTrackremoved = true
			}
		} else {
			// not removing this track
			newQueue = append(newQueue, tr)
		}
	}
	p.playQueue = newQueue
	p.nowPlayingIdx = newNowPlaying
	if isPlayingTrackRemoved {
		if newNowPlaying == len(newQueue) {
			// we had been playing the last track, and removed it
			p.Stop()
		} else {
			p.nowPlayingIdx -= 1 // will be incremented in newtrack callback from player
			p.setTrack(newNowPlaying, false, 0)
		}
		// setNextTrack and onSongChange callbacks will be handled
		// when we receive new track event from player
	} else if isNextPlayingTrackremoved {
		if newNowPlaying < len(newQueue)-1 {
			p.handleNextTrackUpdated()
		} else {
			// no next track to play
			p.setNextTrack(-1)
		}
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
}

func (p *playbackEngine) SetReplayGainOptions(config ReplayGainConfig) {
	rGainPlayer, ok := p.player.(player.ReplayGainPlayer)
	if !ok {
		log.Println("Error: player doesn't support ReplayGain")
		return
	}

	p.replayGainCfg = config
	mode := player.ReplayGainNone
	switch config.Mode {
	case ReplayGainAuto:
		mode = player.ReplayGainTrack
	case ReplayGainTrack:
		mode = player.ReplayGainTrack
	case ReplayGainAlbum:
		mode = player.ReplayGainAlbum
	}

	rGainPlayer.SetReplayGainOptions(player.ReplayGainOptions{
		Mode:            mode,
		PreventClipping: config.PreventClipping,
		PreampGain:      config.PreampGainDB,
	})
}

func (p *playbackEngine) SetReplayGainMode(mode player.ReplayGainMode) {
	rGainPlayer, ok := p.player.(player.ReplayGainPlayer)
	if !ok {
		log.Println("Error: player doesn't support ReplayGain")
		return
	}
	rGainPlayer.SetReplayGainOptions(player.ReplayGainOptions{
		PreventClipping: p.replayGainCfg.PreventClipping,
		PreampGain:      p.replayGainCfg.PreampGainDB,
		Mode:            mode,
	})
}

func (p *playbackEngine) cacheNextTracks() {
	if p.audiocache != nil {
		// fetch up to the 2 next tracks in the queue to the cache
		fetch := make([]AudioCacheRequest, 0, 3)
		// if nothing is playing (index = -1), treat the beginning of the queue as
		// the "currently" playing track, since we're probably about to play it
		npI := max(p.nowPlayingIdx, 0)
		for _, idx := range [3]int{npI, npI + 1, npI + 2} {
			if idx > 0 && idx < len(p.playQueue) {
				item := p.playQueue[idx]
				if item.Metadata().Type == mediaprovider.MediaItemTypeTrack {
					fetch = append(fetch, AudioCacheRequest{
						ID:          p.playQueue[idx].Metadata().ID,
						DownloadURL: p.getMediaURLForIdx(idx),
					})
				}
			}
		}
		id := ""
		if np := p.NowPlaying(); np != nil {
			id = np.Metadata().ID
		}
		p.audiocache.CacheOnly(id, fetch)
	}
}

func (p *playbackEngine) handleOnTrackChange() {
	// scrobble the previous song if needed
	if !p.alreadyScrobbled {
		p.checkScrobble()
	}

	if p.PlaybackStatus().State == player.Playing {
		p.playTimeStopwatch.Start()
	}
	if p.pendingTrackChangeNum < 0 && (p.wasStopped || p.loopMode != LoopOne) {
		p.nowPlayingIdx++
		if p.loopMode == LoopAll && p.nowPlayingIdx == len(p.playQueue) {
			p.nowPlayingIdx = 0 // wrapped around
		}
	} else if p.pendingTrackChangeNum >= 0 {
		p.nowPlayingIdx = p.pendingTrackChangeNum
		p.pendingTrackChangeNum = -1
	}
	// Bounds check for safety against race conditions
	if p.nowPlayingIdx < 0 || p.nowPlayingIdx >= len(p.playQueue) {
		return
	}
	nowPlaying := p.playQueue[p.nowPlayingIdx]
	_, isRadio := nowPlaying.(*mediaprovider.RadioStation)
	p.isRadio = isRadio

	// reset flags
	p.wasStopped = false
	p.alreadyScrobbled = false

	p.curTrackDuration = nowPlaying.Metadata().Duration.Seconds()
	p.sendNowPlayingScrobble() // Must come before invokeOnChangeCallbacks b/c track may immediately be scrobbled
	p.invokeOnSongChangeCallbacks()
	p.handleTimePosUpdate(false)
	p.handleNextTrackUpdated()

	if p.pauseAfterCurrent {
		p.Pause()
		p.SetPauseAfterCurrent(false)
	}

}

func (p *playbackEngine) handleOnStopped() {
	p.playTimeStopwatch.Stop()
	if !p.alreadyScrobbled {
		p.checkScrobble()
	}
	p.stopPollTimePos()
	p.handleTimePosUpdate(false)
	p.invokeOnSongChangeCallbacks()
	p.invokeNoArgCallbacks(p.onStopped)
	p.alreadyScrobbled = false
	p.wasStopped = true
	p.nowPlayingIdx = -1
	p.pauseAfterCurrent = false
}

// to be invoked as soon as the next item in the queue that should play changes
func (p *playbackEngine) handleNextTrackUpdated() {
	p.cacheNextTracks()
	p.needToSetNextTrack = true
	for _, cb := range p.onBeforeSongChange {
		var item mediaprovider.MediaItem
		if idx := p.nextPlayingIndex(); idx >= 0 {
			item = p.playQueue[idx]
		}
		cb(item)
	}
}

func (p *playbackEngine) nextPlayingIndex() int {
	switch p.loopMode {
	case LoopNone:
		if p.nowPlayingIdx >= len(p.playQueue)-1 {
			return -1
		}
		return p.nowPlayingIdx + 1
	case LoopOne:
		return p.nowPlayingIdx
	case LoopAll:
		if p.nowPlayingIdx >= len(p.playQueue)-1 {
			return 0
		}
		return p.nowPlayingIdx + 1
	}
	return -1 // unreached
}

func (p *playbackEngine) setTrack(idx int, next bool, startTime float64) error {
	var item mediaprovider.MediaItem
	var url string
	if idx >= 0 && idx < len(p.playQueue) {
		item = p.playQueue[idx]
		url = p.getMediaURLForIdx(idx)
	}
	track, isTrack := item.(*mediaprovider.Track)
	if p.audiocache != nil && isTrack {
		p.audiocache.CacheFile(item.Metadata().ID, p.getMediaURLForIdx(idx))
	}

	if urlP, ok := p.player.(player.URLPlayer); ok {
		var meta mediaprovider.MediaItemMetadata
		if item != nil {
			meta = item.Metadata()
			if isTrack && p.audiocache != nil {
				if filepath := p.audiocache.PathForCachedFile(track.ID); filepath != "" {
					url = filepath
				}
			}
			if url == "" {
				return errors.New("no stream URL")
			}
		}
		if next {
			return urlP.SetNextFile(url, meta)
		}
		return urlP.PlayFile(url, meta, startTime)
	} else if trP, ok := p.player.(player.TrackPlayer); ok {
		var track *mediaprovider.Track
		if item != nil {
			track, ok = item.(*mediaprovider.Track)
			if !ok {
				return errors.New("cannot play non-Track media item with TrackPlayer")
			}
		}
		if next {
			return trP.SetNextTrack(track)
		}
		// If player supports queue loading and has a loaded queue, use PlayQueueIndex
		if qp, ok := p.player.(player.QueuePlayer); ok && qp.IsQueueLoaded() && idx >= 0 && idx < len(p.playQueue) {
			return qp.PlayQueueIndex(idx)
		}
		return trP.PlayTrack(track, startTime)
	}
	panic("Unsupported player type")
}

func (p *playbackEngine) getMediaURLForIdx(idx int) string {
	var url string
	item := p.playQueue[idx]
	if tr, ok := item.(*mediaprovider.Track); ok {
		var ts *mediaprovider.TranscodeSettings
		if p.transcodeCfg.RequestTranscode {
			ts = &mediaprovider.TranscodeSettings{
				Codec:       p.transcodeCfg.Codec,
				BitRateKBPS: p.transcodeCfg.MaxBitRateKBPS,
			}
		}
		url, _ = p.sm.Server.GetStreamURL(tr.ID, ts, p.transcodeCfg.ForceRawFile)
	} else {
		url = item.(*mediaprovider.RadioStation).StreamURL
	}
	return url
}

func (p *playbackEngine) setNextTrack(idx int) error {
	return p.setTrack(idx, true, 0)
}

// call BEFORE updating p.nowPlayingIdx
func (p *playbackEngine) checkScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	track, ok := p.playQueue[p.nowPlayingIdx].(*mediaprovider.Track)
	if !ok {
		return // radio stations are not scrobbled
	}

	playDur := p.playTimeStopwatch.Elapsed()
	if playDur.Seconds() < 0.1 || p.curTrackDuration < 0.1 {
		return
	}
	pcnt := playDur.Seconds() / p.curTrackDuration * 100
	timeThresholdMet := p.scrobbleCfg.ThresholdTimeSeconds >= 0 &&
		playDur.Seconds() >= float64(p.scrobbleCfg.ThresholdTimeSeconds)

	var submission bool
	server := p.sm.Server
	if server.ClientDecidesScrobble() && (timeThresholdMet || pcnt >= float64(p.scrobbleCfg.ThresholdPercent)) {
		track.PlayCount += 1
		p.lastScrobbled = track
		submission = true
	}
	go server.TrackEndedPlayback(track.ID, int(p.latestTrackPosition), submission)
	p.latestTrackPosition = 0
	p.playTimeStopwatch.Reset()
}

func (p *playbackEngine) sendNowPlayingScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	track, ok := p.playQueue[p.nowPlayingIdx].(*mediaprovider.Track)
	if !ok {
		return // radio stations are not scrobbled
	}

	server := p.sm.Server
	if !server.ClientDecidesScrobble() {
		// server will count track as scrobbled as soon as it starts playing
		p.lastScrobbled = track
		track.PlayCount += 1
	}
	go p.sm.Server.TrackBeganPlayback(track.ID)
}

// creates a deep copy of the track info so that we can maintain our own state
// (play count increases, favorite, and rating) without messing up other views' track models
func deepCopyMediaItemSlice(tracks []mediaprovider.MediaItem) []mediaprovider.MediaItem {
	newTracks := make([]mediaprovider.MediaItem, len(tracks))
	for i, tr := range tracks {
		newTracks[i] = tr.Copy()
	}
	return newTracks
}

func copyTrackSliceToMediaItemSlice(tracks []*mediaprovider.Track) []mediaprovider.MediaItem {
	newTracks := make([]mediaprovider.MediaItem, len(tracks))
	for i, tr := range tracks {
		newTracks[i] = tr.Copy()
	}
	return newTracks
}

func (p *playbackEngine) invokeOnSongChangeCallbacks() {
	if p.callbacksDisabled {
		return
	}
	for _, cb := range p.onSongChange {
		cb(p.NowPlaying(), p.lastScrobbled)
	}
	p.lastScrobbled = nil
}

func (pm *playbackEngine) invokeNoArgCallbacks(cbs []func()) {
	if pm.callbacksDisabled {
		return
	}
	for _, cb := range cbs {
		cb()
	}
}

func (p *playbackEngine) startPollTimePos() {
	// Stop any existing polling goroutine first to prevent leaks
	p.stopPollTimePos()

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancelPollPos = cancel
	pollFrequency := 250 * time.Millisecond
	if p.playbackCfg.UseWaveformSeekbar {
		pollFrequency = 100 * time.Millisecond
	}
	pollingTick := time.NewTicker(pollFrequency)

	go func() {
		for {
			select {
			case <-ctx.Done():
				pollingTick.Stop()
				return
			case <-pollingTick.C:
				p.handleTimePosUpdate(false)
			}
		}
	}()
}

func (p *playbackEngine) stopPollTimePos() {
	if p.cancelPollPos != nil {
		p.cancelPollPos()
		p.cancelPollPos = nil
	}
}

func (p *playbackEngine) handleTimePosUpdate(seeked bool) {
	s := p.PlaybackStatus()
	var meta mediaprovider.MediaItemMetadata
	if np := p.NowPlaying(); np != nil {
		meta = np.Metadata()
	}
	isNearEnd := meta.Type != mediaprovider.MediaItemTypeRadioStation && s.TimePos > meta.Duration.Seconds()-10
	if p.needToSetNextTrack && isNearEnd {
		p.needToSetNextTrack = false
		p.setNextTrack(p.nextPlayingIndex())
	}
	if p.callbacksDisabled {
		return
	}
	if s.TimePos > p.latestTrackPosition {
		p.latestTrackPosition = s.TimePos
	}
	duration := s.Duration
	if p.isRadio {
		// MPV reports buffered duration - we don't want to show this
		duration = 0
	}
	for _, cb := range p.onPlayTimeUpdate {
		cb(s.TimePos, duration, seeked)
	}
}
