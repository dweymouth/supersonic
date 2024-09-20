package backend

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"slices"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/dweymouth/supersonic/sharedutil"
)

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

type playbackEngine struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	sm            *ServerManager
	player        player.BasePlayer

	playTimeStopwatch   util.Stopwatch
	curTrackDuration    float64
	latestTrackPosition float64 // cleared by checkScrobble
	callbacksDisabled   bool

	playQueue    []mediaprovider.MediaItem
	shuffleOrder []int
	// position in the play queue order. If shuffling, the actual
	// track index is p.shuffleOrder[p.playQueuePosition]
	playQueuePosition int
	shuffle           bool

	isRadio                    bool
	wasStopped                 bool // true iff player was stopped before handleOnTrackChange invocation
	noIncrementNextTrackChange bool // true iff the nowPlayingIndex should not be incremente don the next onTrackChange
	loopMode                   LoopMode

	// to pass to onSongChange listeners; clear once listeners have been called
	lastScrobbled *mediaprovider.Track
	scrobbleCfg   *ScrobbleConfig
	transcodeCfg  *TranscodingConfig
	replayGainCfg ReplayGainConfig

	// registered callbacks
	onSongChange     []func(nowPlaying mediaprovider.MediaItem, justScrobbledIfAny *mediaprovider.Track)
	onPlayTimeUpdate []func(float64, float64, bool)
	onLoopModeChange []func(LoopMode)
	onVolumeChange   []func(int)
	onSeek           []func()
	onPaused         []func()
	onStopped        []func()
	onPlaying        []func()
	onPlayerChange   []func()
	onQueueChange    []func()
}

func NewPlaybackEngine(
	ctx context.Context,
	s *ServerManager,
	p player.BasePlayer,
	scrobbleCfg *ScrobbleConfig,
	transcodeCfg *TranscodingConfig,
) *playbackEngine {
	// clamp to 99% to avoid any possible rounding issues
	scrobbleCfg.ThresholdPercent = clamp(scrobbleCfg.ThresholdPercent, 0, 99)
	pm := &playbackEngine{
		ctx:               ctx,
		sm:                s,
		player:            p,
		scrobbleCfg:       scrobbleCfg,
		transcodeCfg:      transcodeCfg,
		playQueuePosition: -1,
		wasStopped:        true,
	}
	p.OnTrackChange(pm.handleOnTrackChange)
	p.OnSeek(func() {
		pm.doUpdateTimePos(true)
		pm.invokeNoArgCallbacks(pm.onSeek)
	})
	p.OnStopped(pm.handleOnStopped)
	p.OnPaused(func() {
		pm.playTimeStopwatch.Stop()
		pm.stopPollTimePos()
		pm.invokeNoArgCallbacks(pm.onPaused)
	})
	p.OnPlaying(func() {
		pm.playTimeStopwatch.Start()
		pm.startPollTimePos()
		pm.invokeNoArgCallbacks(pm.onPlaying)
	})

	s.OnLogout(func() {
		pm.StopAndClearPlayQueue()
	})

	return pm
}

// PlayTrackAt plays the track at the given index in the play queue
func (p *playbackEngine) PlayTrackAt(idx int) error {
	if idx < 0 || idx >= len(p.playQueue) {
		return errors.New("track index out of range")
	}
	p.noIncrementNextTrackChange = true
	err := p.setTrack(idx, false)
	if err == nil {
		if p.shuffle {
			p.playQueuePosition = slices.Index(p.shuffleOrder, idx)
		} else {
			p.playQueuePosition = idx
		}
	}
	return err
}

// plays the track at the given queue position (NOT track index)
// if shuffling, the actual track index is p.shuffleOrder[pos]
func (p *playbackEngine) playAtQueuePosition(pos int) error {
	idx := p.playQueuePositionToTrackIdx(pos)
	p.noIncrementNextTrackChange = true
	err := p.setTrack(idx, false)
	if err == nil {
		p.playQueuePosition = pos
	}
	return err
}

func (p *playbackEngine) playQueuePositionToTrackIdx(pos int) int {
	idx := pos
	if p.shuffle {
		idx = p.shuffleOrder[pos]
	}
	return idx
}

// NowPlaying returns the curently playing media item, if any.
func (p *playbackEngine) NowPlaying() mediaprovider.MediaItem {
	if p.playQueuePosition < 0 || len(p.playQueue) == 0 || p.player.GetStatus().State == player.Stopped {
		return nil
	}
	return p.playQueue[p.NowPlayingIndex()]
}

// NowPlayingIndex returns the index in the play queue of the currently playing track.
func (p *playbackEngine) NowPlayingIndex() int {
	if p.shuffle {
		return p.shuffleOrder[p.playQueuePosition]
	}
	return p.playQueuePosition
}

func (p *playbackEngine) SetLoopMode(loopMode LoopMode) {
	p.loopMode = loopMode
	if p.playQueuePosition >= 0 {
		p.setNextTrackBasedOnLoopMode(true)
	}

	for _, cb := range p.onLoopModeChange {
		cb(loopMode)
	}
}

func (p *playbackEngine) GetLoopMode() LoopMode {
	return p.loopMode
}

func (p *playbackEngine) PlayerStatus() player.Status {
	return p.player.GetStatus()
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

func (p *playbackEngine) SeekFwdBackN(n int) error {
	pos := p.playQueuePosition
	if pos == -1 {
		return nil // stopped
	}
	if n < 0 && p.player.GetStatus().TimePos > 3 {
		n += 1 // first seek back is just seek to beginning of current
	}
	if n == 0 || (pos == 0 && n < 0) {
		return p.player.SeekSeconds(0) // seek back in current song
	}
	lastPos := len(p.playQueue) - 1
	if pos == lastPos && n > 0 {
		return nil // already on last position, nothing to seek next to
	}
	newPos := minInt(lastPos, maxInt(0, pos+n))
	return p.playAtQueuePosition(newPos)
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
	return p.player.Stop()
}

func (p *playbackEngine) Pause() error {
	return p.player.Pause()
}

func (p *playbackEngine) Continue() error {
	if p.PlayerStatus().State == player.Stopped {
		return p.PlayTrackAt(0)
	}
	return p.player.Continue()
}

// SetShuffle sets whether shuffle order is enabled
func (p *playbackEngine) SetShuffle(shuffle bool) {
	if p.shuffle == shuffle {
		return
	}
	p.shuffle = shuffle
	if shuffle {
		p.generateShuffleOrder()
	} else {
		p.shuffleOrder = nil
	}
}

// IsShuffle returns whether shuffle order is enabled
func (p *playbackEngine) IsShuffle() bool {
	return p.shuffle
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
	if insertQueueMode == Replace {
		p.player.Stop()
		p.playQueuePosition = -1
		p.playQueue = nil
	}
	needToSetNext := len(items) > 0 && (insertQueueMode == InsertNext || (insertQueueMode == Append && p.playQueuePosition == len(p.playQueue)-1))

	if shuffle {
		rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	}

	insertIdx := len(p.playQueue)
	if insertQueueMode == InsertNext {
		insertIdx = p.playQueuePosition + 1
	}
	p.playQueue = append(p.playQueue[:insertIdx], append(items, p.playQueue[insertIdx:]...)...)

	if needToSetNext {
		p.setNextTrack(p.playQueuePosition + 1)
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
	return nil
}

func (p *playbackEngine) LoadRadioStation(radio *mediaprovider.RadioStation, insertMode InsertQueueMode) {
	if insertMode == Replace {
		p.player.Stop()
		p.playQueuePosition = -1
		p.playQueue = nil
	}
	needToSetNext := insertMode == InsertNext || (insertMode == Append && p.playQueuePosition == len(p.playQueue)-1)
	insertIdx := len(p.playQueue)
	if insertMode == InsertNext {
		insertIdx = p.playQueuePosition + 1
	}
	new := make([]mediaprovider.MediaItem, len(p.playQueue)+1)
	firstHalf := p.playQueue[:insertIdx]
	copy(new, firstHalf)
	new[len(firstHalf)] = radio
	copy(new[len(firstHalf)+1:], p.playQueue[insertIdx:])
	p.playQueue = new

	if needToSetNext {
		p.setNextTrack(p.playQueuePosition + 1)
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
}

// Stop playback and clear the play queue.
func (p *playbackEngine) StopAndClearPlayQueue() {
	changed := len(p.playQueue) > 0
	p.player.Stop()
	p.playQueue = nil
	p.shuffleOrder = nil
	p.playQueuePosition = -1
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
	newQueue := deepCopyMediaItemSlice(items)
	newNowPlayingIdx := -1
	if p.playQueuePosition >= 0 {
		nowPlayingID := p.NowPlaying().Metadata().ID
		for i, tr := range newQueue {
			if tr.Metadata().ID == nowPlayingID {
				newNowPlayingIdx = i
				break
			}
		}
	}

	if p.playQueuePosition >= 0 && newNowPlayingIdx == -1 {
		return p.Stop() // was playing a track that is no longer in the queue
	}
	p.playQueue = newQueue
	if p.shuffle {
		p.generateShuffleOrder()
		// TODO: set p.playQueuePosition to first idx of nowPlayingID in shuffle order
	} else {
		p.playQueuePosition = newNowPlayingIdx
	}
	if p.playQueuePosition >= 0 { // need to update next track
		p.setNextTrackAfterQueueUpdate()
	}

	p.invokeNoArgCallbacks(p.onQueueChange)
	return nil
}

func (p *playbackEngine) RemoveTracksFromQueue(idxs []int) {
	// TODO: this logic is totally broken when shuffling

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
			} else if nowPlaying >= 0 && i == nowPlaying+1 {
				isNextPlayingTrackremoved = true
			}
		} else {
			// not removing this track
			newQueue = append(newQueue, tr)
		}
	}
	p.playQueue = newQueue
	p.playQueuePosition = newNowPlaying
	if isPlayingTrackRemoved {
		if newNowPlaying == len(newQueue) {
			// we had been playing the last track, and removed it
			p.Stop()
		} else {
			p.playQueuePosition -= 1 // will be incremented in newtrack callback from player
			p.setTrack(newNowPlaying, false)
		}
		// setNextTrack and onSongChange callbacks will be handled
		// when we receive new track event from player
	} else if isNextPlayingTrackremoved {
		if newNowPlaying < len(newQueue)-1 {
			p.setNextTrack(p.playQueuePosition + 1)
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

// generates a new randomized shuffle order for the tracks in the play queue
func (p *playbackEngine) generateShuffleOrder() {
	p.shuffleOrder = make([]int, len(p.playQueue))
	for i := range p.shuffleOrder {
		p.shuffleOrder[i] = i
	}
	rand.Shuffle(len(p.shuffleOrder), func(i, j int) {
		p.shuffleOrder[i], p.shuffleOrder[j] = p.shuffleOrder[j], p.shuffleOrder[i]
	})
}

func (p *playbackEngine) handleOnTrackChange() {
	p.checkScrobble() // scrobble the previous song if needed
	if p.player.GetStatus().State == player.Playing {
		p.playTimeStopwatch.Start()
	}
	if !p.noIncrementNextTrackChange && (p.wasStopped || p.loopMode != LoopOne) {
		p.playQueuePosition++
		if p.loopMode == LoopAll && p.playQueuePosition == len(p.playQueue) {
			p.playQueuePosition = 0 // wrapped around
		}
	}
	p.noIncrementNextTrackChange = false
	nowPlaying := p.NowPlaying()
	_, isRadio := nowPlaying.(*mediaprovider.RadioStation)
	p.isRadio = isRadio
	p.wasStopped = false
	p.curTrackDuration = float64(nowPlaying.Metadata().Duration)
	p.sendNowPlayingScrobble() // Must come before invokeOnChangeCallbacks b/c track may immediately be scrobbled
	p.invokeOnSongChangeCallbacks()
	p.doUpdateTimePos(false)
	p.setNextTrackBasedOnLoopMode(false)
}

func (p *playbackEngine) handleOnStopped() {
	p.playTimeStopwatch.Stop()
	p.checkScrobble()
	p.stopPollTimePos()
	p.doUpdateTimePos(false)
	p.invokeOnSongChangeCallbacks()
	p.invokeNoArgCallbacks(p.onStopped)
	p.wasStopped = true
	p.playQueuePosition = -1
}

func (p *playbackEngine) setNextTrackBasedOnLoopMode(onLoopModeChange bool) {
	switch p.loopMode {
	case LoopNone:
		if p.playQueuePosition < len(p.playQueue)-1 {
			p.setNextTrack(p.playQueuePosition + 1)
		} else if onLoopModeChange {
			// prev was LoopOne - need to erase next track
			p.setNextTrack(-1)
		}
	case LoopOne:
		p.setNextTrack(p.playQueuePosition)
	case LoopAll:
		if p.playQueuePosition >= len(p.playQueue)-1 {
			p.setNextTrack(0)
		} else if !onLoopModeChange {
			// if onloopmodechange, prev mode was LoopNone and next track is already set
			p.setNextTrack(p.playQueuePosition + 1)
		}
	}
}

func (p *playbackEngine) setNextTrackAfterQueueUpdate() {
	switch p.loopMode {
	case LoopNone:
		if p.playQueuePosition < len(p.playQueue)-1 {
			p.setNextTrack(p.playQueuePosition + 1)
		} else {
			// need to erase next track
			p.setNextTrack(-1)
		}
	case LoopOne:
		p.setNextTrack(p.playQueuePosition)
	case LoopAll:
		if p.playQueuePosition >= len(p.playQueue)-1 {
			p.setNextTrack(0)
		} else {
			p.setNextTrack(p.playQueuePosition + 1)
		}
	}
}

func (p *playbackEngine) setTrack(idx int, next bool) error {
	if urlP, ok := p.player.(player.URLPlayer); ok {
		url := ""
		if idx >= 0 {
			var err error
			item := p.playQueue[idx]
			if tr, ok := item.(*mediaprovider.Track); ok {
				url, err = p.sm.Server.GetStreamURL(tr.ID, p.transcodeCfg.ForceRawFile)
			} else {
				url = item.(*mediaprovider.RadioStation).StreamURL
			}
			if err != nil {
				return err
			}
		}
		if next {
			return urlP.SetNextFile(url)
		}
		return urlP.PlayFile(url)
	} else if trP, ok := p.player.(player.TrackPlayer); ok {
		var track *mediaprovider.Track
		if idx >= 0 {
			track, ok = p.playQueue[idx].(*mediaprovider.Track)
			if !ok {
				return errors.New("cannot play non-Track media item with TrackPlayer")
			}
		}
		if next {
			return trP.SetNextTrack(track)
		}
		return trP.PlayTrack(track)
	}
	panic("Unsupported player type")
}

func (p *playbackEngine) setNextTrack(idx int) error {
	return p.setTrack(idx, true)
}

// call BEFORE updating p.nowPlayingIdx
func (p *playbackEngine) checkScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.playQueuePosition < 0 {
		return
	}
	track, ok := p.NowPlaying().(*mediaprovider.Track)
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
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.playQueuePosition < 0 {
		return
	}
	track, ok := p.playQueue[p.playQueuePosition].(*mediaprovider.Track)
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
	ctx, cancel := context.WithCancel(p.ctx)
	p.cancelPollPos = cancel
	pollingTick := time.NewTicker(250 * time.Millisecond)

	go func() {
		for {
			select {
			case <-ctx.Done():
				pollingTick.Stop()
				return
			case <-pollingTick.C:
				p.doUpdateTimePos(false)
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

func (p *playbackEngine) doUpdateTimePos(seeked bool) {
	if p.callbacksDisabled {
		return
	}
	s := p.player.GetStatus()
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
