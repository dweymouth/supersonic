package backend

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/sharedutil"
)

var (
	ReplayGainNone  = player.ReplayGainNone.String()
	ReplayGainAlbum = player.ReplayGainAlbum.String()
	ReplayGainTrack = player.ReplayGainTrack.String()
	ReplayGainAuto  = "Auto"
)

// The playback loop mode (LoopNone, LoopAll, LoopOne).
type LoopMode int

const (
	LoopNone LoopMode = iota
	LoopAll
	LoopOne
)

// A high-level MediaProvider-aware playback engine, serves as an
// intermediary between the frontend and various Player backends.
type PlaybackManager struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	sm            *ServerManager
	player        player.BasePlayer

	playTimeStopwatch   util.Stopwatch
	curTrackTime        float64
	latestTrackPosition float64 // cleared by checkScrobble
	callbacksDisabled   bool

	playQueue     []*mediaprovider.Track
	nowPlayingIdx int
	wasStopped    bool // true iff player was stopped before handleOnTrackChange invocation
	loopMode      LoopMode

	// to pass to onSongChange listeners; clear once listeners have been called
	lastScrobbled *mediaprovider.Track
	scrobbleCfg   *ScrobbleConfig
	transcodeCfg  *TranscodingConfig
	replayGainCfg ReplayGainConfig

	// registered callbacks
	onSongChange     []func(nowPlaying, justScrobbledIfAny *mediaprovider.Track)
	onPlayTimeUpdate []func(float64, float64)
	onLoopModeChange []func(LoopMode)
	onVolumeChange   []func(int)
	onSeek           []func()
	onPaused         []func()
	onStopped        []func()
	onPlaying        []func()
	onPlayerChange   []func()
}

func NewPlaybackManager(
	ctx context.Context,
	s *ServerManager,
	p player.BasePlayer,
	scrobbleCfg *ScrobbleConfig,
	transcodeCfg *TranscodingConfig,
) *PlaybackManager {
	// clamp to 99% to avoid any possible rounding issues
	scrobbleCfg.ThresholdPercent = clamp(scrobbleCfg.ThresholdPercent, 0, 99)
	pm := &PlaybackManager{
		ctx:           ctx,
		sm:            s,
		player:        p,
		scrobbleCfg:   scrobbleCfg,
		transcodeCfg:  transcodeCfg,
		nowPlayingIdx: -1,
		wasStopped:    true,
	}
	p.OnTrackChange(pm.handleOnTrackChange)
	p.OnSeek(func() {
		pm.doUpdateTimePos()
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

func (p *PlaybackManager) CurrentPlayer() player.BasePlayer {
	return p.player
}

func (p *PlaybackManager) OnPlayerChange(cb func()) {
	p.onPlayerChange = append(p.onPlayerChange, cb)
}

func (p *PlaybackManager) IsSeeking() bool {
	return p.player.IsSeeking()
}

// Should only be called before quitting.
// Disables playback state callbacks being sent
func (p *PlaybackManager) DisableCallbacks() {
	p.callbacksDisabled = true
}

// Gets the curently playing song, if any.
func (p *PlaybackManager) NowPlaying() *mediaprovider.Track {
	if p.nowPlayingIdx < 0 || len(p.playQueue) == 0 || p.player.GetStatus().State == player.Stopped {
		return nil
	}
	return p.playQueue[p.nowPlayingIdx]
}

func (p *PlaybackManager) NowPlayingIndex() int {
	return int(p.nowPlayingIdx)
}

// Sets a callback that is notified whenever a new song begins playing.
func (p *PlaybackManager) OnSongChange(cb func(nowPlaying *mediaprovider.Track, justScrobbledIfAny *mediaprovider.Track)) {
	p.onSongChange = append(p.onSongChange, cb)
}

// Registers a callback that is notified whenever the play time should be updated.
func (p *PlaybackManager) OnPlayTimeUpdate(cb func(float64, float64)) {
	p.onPlayTimeUpdate = append(p.onPlayTimeUpdate, cb)
}

// Registers a callback that is notified whenever the loop mode changes.
func (p *PlaybackManager) OnLoopModeChange(cb func(LoopMode)) {
	p.onLoopModeChange = append(p.onLoopModeChange, cb)
}

// Registers a callback that is notified whenever the volume changes.
func (p *PlaybackManager) OnVolumeChange(cb func(int)) {
	p.onVolumeChange = append(p.onVolumeChange, cb)
}

// Registers a callback that is notified whenever the player has been seeked.
func (p *PlaybackManager) OnSeek(cb func()) {
	p.onSeek = append(p.onSeek, cb)
}

// Registers a callback that is notified whenever the player has been paused.
func (p *PlaybackManager) OnPaused(cb func()) {
	p.onPaused = append(p.onPaused, cb)
}

// Registers a callback that is notified whenever the player is stopped.
func (p *PlaybackManager) OnStopped(cb func()) {
	p.onStopped = append(p.onStopped, cb)
}

// Registers a callback that is notified whenever the player begins playing.
func (p *PlaybackManager) OnPlaying(cb func()) {
	p.onPlaying = append(p.onPlaying, cb)
}

// Loads the specified album into the play queue.
func (p *PlaybackManager) LoadAlbum(albumID string, appendToQueue bool, shuffle bool) error {
	album, err := p.sm.Server.GetAlbum(albumID)
	if err != nil {
		return err
	}
	return p.LoadTracks(album.Tracks, appendToQueue, shuffle)
}

// Loads the specified playlist into the play queue.
func (p *PlaybackManager) LoadPlaylist(playlistID string, appendToQueue bool, shuffle bool) error {
	playlist, err := p.sm.Server.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	return p.LoadTracks(playlist.Tracks, appendToQueue, shuffle)
}

// Load tracks into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *PlaybackManager) LoadTracks(tracks []*mediaprovider.Track, appendToQueue, shuffle bool) error {
	if !appendToQueue {
		p.player.Stop()
		p.nowPlayingIdx = -1
		p.playQueue = nil
	}
	needToSetNext := appendToQueue && len(tracks) > 0 && p.nowPlayingIdx == len(p.playQueue)-1

	newTracks := p.deepCopyTrackSlice(tracks)
	if shuffle {
		rand.Shuffle(len(newTracks), func(i, j int) { newTracks[i], newTracks[j] = newTracks[j], newTracks[i] })
	}
	p.playQueue = append(p.playQueue, newTracks...)

	if needToSetNext {
		p.setNextTrack(p.nowPlayingIdx + 1)
	}
	return nil
}

// Replaces the play queue with the given set of tracks.
// Does not stop playback if the currently playing track is in the new queue,
// but updates the now playing index to point to the first instance of the track in the new queue.
func (p *PlaybackManager) UpdatePlayQueue(tracks []*mediaprovider.Track) error {
	newQueue := p.deepCopyTrackSlice(tracks)
	newNowPlayingIdx := -1
	if p.nowPlayingIdx >= 0 {
		nowPlayingID := p.playQueue[p.nowPlayingIdx].ID
		for i, tr := range newQueue {
			if tr.ID == nowPlayingID {
				newNowPlayingIdx = i
				break
			}
		}
	}

	p.playQueue = newQueue
	if p.nowPlayingIdx >= 0 && newNowPlayingIdx == -1 {
		return p.Stop()
	}
	needToUpdateNext := p.nowPlayingIdx >= 0
	p.nowPlayingIdx = newNowPlayingIdx
	if needToUpdateNext {
		p.setNextTrackAfterQueueUpdate()
	}

	return nil
}

func (p *PlaybackManager) PlayAlbum(albumID string, firstTrack int, shuffle bool) error {
	if err := p.LoadAlbum(albumID, false, shuffle); err != nil {
		return err
	}
	if p.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainAlbum)
	}
	return p.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int, shuffle bool) error {
	if err := p.LoadPlaylist(playlistID, false, shuffle); err != nil {
		return err
	}
	if p.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	return p.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayTrack(trackID string) error {
	tr, err := p.sm.Server.GetTrack(trackID)
	if err != nil {
		return err
	}
	p.LoadTracks([]*mediaprovider.Track{tr}, false, false)
	if p.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	return p.PlayFromBeginning()
}

func (p *PlaybackManager) PlayFromBeginning() error {
	return p.PlayTrackAt(0)
}

func (p *PlaybackManager) PlayTrackAt(idx int) error {
	if idx < 0 || idx >= len(p.playQueue) {
		return errors.New("track index out of range")
	}
	p.nowPlayingIdx = idx - 1
	return p.setTrack(idx, false)
}

func (p *PlaybackManager) PlayRandomSongs(genreName string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.sm.Server.GetRandomTracks(genreName, 100)
	})
}

func (p *PlaybackManager) PlaySimilarSongs(id string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.sm.Server.GetSimilarTracks(id, 100)
	})
}

func (p *PlaybackManager) fetchAndPlayTracks(fetchFn func() ([]*mediaprovider.Track, error)) {
	if songs, err := fetchFn(); err != nil {
		log.Printf("error fetching tracks: %s", err.Error())
	} else {
		p.LoadTracks(songs, false, false)
		if p.replayGainCfg.Mode == ReplayGainAuto {
			p.SetReplayGainMode(player.ReplayGainTrack)
		}
		p.PlayFromBeginning()
	}
}

func (p *PlaybackManager) GetPlayQueue() []*mediaprovider.Track {
	return p.deepCopyTrackSlice(p.playQueue)
}

// Any time the user changes the favorite status of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *PlaybackManager) OnTrackFavoriteStatusChanged(id string, fav bool) {
	if tr := sharedutil.FindTrackByID(id, p.playQueue); tr != nil {
		tr.Favorite = fav
	}
}

// Any time the user changes the rating of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *PlaybackManager) OnTrackRatingChanged(id string, rating int) {
	if tr := sharedutil.FindTrackByID(id, p.playQueue); tr != nil {
		tr.Rating = rating
	}
}

func (p *PlaybackManager) RemoveTracksFromQueue(trackIDs []string) {
	newQueue := make([]*mediaprovider.Track, 0, len(p.playQueue)-len(trackIDs))
	idSet := sharedutil.ToSet(trackIDs)
	isPlayingTrackRemoved := false
	isNextPlayingTrackremoved := false
	nowPlaying := p.NowPlayingIndex()
	newNowPlaying := nowPlaying
	for i, tr := range p.playQueue {
		if _, ok := idSet[tr.ID]; ok {
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
	p.nowPlayingIdx = newNowPlaying
	if isPlayingTrackRemoved {
		if newNowPlaying == len(newQueue) {
			// we had been playing the last track, and removed it
			p.Stop()
		} else {
			p.nowPlayingIdx -= 1 // will be incremented in newtrack callback from player
			p.setTrack(newNowPlaying, false)
		}
		// setNextTrack and onSongChange callbacks will be handled
		// when we receive new track event from player
	} else if isNextPlayingTrackremoved {
		if newNowPlaying < len(newQueue)-1 {
			p.setNextTrack(p.nowPlayingIdx + 1)
		} else {
			// no next track to play
			p.setNextTrack(-1)
		}
	}
}

// Stop playback and clear the play queue.
func (p *PlaybackManager) StopAndClearPlayQueue() {
	p.player.Stop()
	p.doUpdateTimePos()
	p.playQueue = nil
	p.nowPlayingIdx = -1
}

func (p *PlaybackManager) SetReplayGainOptions(config ReplayGainConfig) {
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

func (p *PlaybackManager) SetReplayGainMode(mode player.ReplayGainMode) {
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

// Changes the loop mode of the player to the next one.
// Useful for toggling UI elements, to change modes without knowing the current player mode.
func (p *PlaybackManager) SetNextLoopMode() {
	switch p.loopMode {
	case LoopNone:
		p.SetLoopMode(LoopAll)
	case LoopAll:
		p.SetLoopMode(LoopOne)
	case LoopOne:
		p.SetLoopMode(LoopNone)

	}
}

func (p *PlaybackManager) SetLoopMode(loopMode LoopMode) {
	p.loopMode = loopMode
	if p.nowPlayingIdx >= 0 {
		p.setNextTrackBasedOnLoopMode(true)
	}

	for _, cb := range p.onLoopModeChange {
		cb(loopMode)
	}
}

func (p *PlaybackManager) GetLoopMode() LoopMode {
	return p.loopMode
}

func (p *PlaybackManager) PlayerStatus() player.Status {
	return p.player.GetStatus()
}

func (p *PlaybackManager) SetVolume(vol int) error {
	vol = clamp(vol, 0, 100)
	if err := p.player.SetVolume(vol); err != nil {
		return err
	}
	for _, cb := range p.onVolumeChange {
		cb(vol)
	}
	return nil
}

func (p *PlaybackManager) Volume() int {
	return p.player.GetVolume()
}

func (p *PlaybackManager) SeekNext() error {
	if p.CurrentPlayer().GetStatus().State == player.Stopped {
		return nil
	}
	return p.PlayTrackAt(p.nowPlayingIdx + 1)
}

func (p *PlaybackManager) SeekBackOrPrevious() error {
	if p.nowPlayingIdx == 0 || p.player.GetStatus().TimePos > 3 {
		return p.player.SeekSeconds(0)
	}
	return p.PlayTrackAt(p.nowPlayingIdx - 1)
}

// Seek to given absolute position in the current track by seconds.
func (p *PlaybackManager) SeekSeconds(sec float64) error {
	return p.player.SeekSeconds(sec)
}

// Seek to a fractional position in the current track [0..1]
func (p *PlaybackManager) SeekFraction(fraction float64) error {
	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}
	target := p.curTrackTime * fraction
	return p.player.SeekSeconds(target)
}

func (p *PlaybackManager) Stop() error {
	return p.player.Stop()
}

func (p *PlaybackManager) Pause() error {
	return p.player.Pause()
}

func (p *PlaybackManager) Continue() error {
	if p.player.GetStatus().State == player.Stopped {
		return p.PlayFromBeginning()
	}
	return p.player.Continue()
}

func (p *PlaybackManager) PlayPause() error {
	switch p.player.GetStatus().State {
	case player.Playing:
		return p.player.Pause()
	case player.Paused:
		return p.player.Continue()
	case player.Stopped:
		return p.PlayFromBeginning()
	}
	return errors.New("unreached - invalid player state")
}

func (p *PlaybackManager) handleOnTrackChange() {
	p.checkScrobble() // scrobble the previous song if needed
	if p.player.GetStatus().State == player.Playing {
		p.playTimeStopwatch.Start()
	}
	if p.wasStopped || p.loopMode != LoopOne {
		p.nowPlayingIdx++
		if p.loopMode == LoopAll && p.nowPlayingIdx == len(p.playQueue) {
			p.nowPlayingIdx = 0 // wrapped around
		}
	}
	p.wasStopped = false
	p.curTrackTime = float64(p.playQueue[p.nowPlayingIdx].Duration)
	p.sendNowPlayingScrobble() // Must come before invokeOnChangeCallbacks b/c track may immediately be scrobbled
	p.invokeOnSongChangeCallbacks()
	p.doUpdateTimePos()
	p.setNextTrackBasedOnLoopMode(false)
}

func (p *PlaybackManager) handleOnStopped() {
	p.playTimeStopwatch.Stop()
	p.checkScrobble()
	p.stopPollTimePos()
	p.doUpdateTimePos()
	p.invokeOnSongChangeCallbacks()
	p.invokeNoArgCallbacks(p.onStopped)
	p.wasStopped = true
	p.nowPlayingIdx = -1
}

func (p *PlaybackManager) setNextTrackBasedOnLoopMode(onLoopModeChange bool) {
	switch p.loopMode {
	case LoopNone:
		if p.nowPlayingIdx < len(p.playQueue)-1 {
			p.setNextTrack(p.nowPlayingIdx + 1)
		} else if onLoopModeChange {
			// prev was LoopOne - need to erase next track
			p.setNextTrack(-1)
		}
	case LoopOne:
		p.setNextTrack(p.nowPlayingIdx)
	case LoopAll:
		if p.nowPlayingIdx >= len(p.playQueue)-1 {
			p.setNextTrack(0)
		} else if !onLoopModeChange {
			// if onloopmodechange, prev mode was LoopNone and next track is already set
			p.setNextTrack(p.nowPlayingIdx + 1)
		}
	}
}

func (p *PlaybackManager) setNextTrackAfterQueueUpdate() {
	switch p.loopMode {
	case LoopNone:
		if p.nowPlayingIdx < len(p.playQueue)-1 {
			p.setNextTrack(p.nowPlayingIdx + 1)
		} else {
			// need to erase next track
			p.setNextTrack(-1)
		}
	case LoopOne:
		p.setNextTrack(p.nowPlayingIdx)
	case LoopAll:
		if p.nowPlayingIdx >= len(p.playQueue)-1 {
			p.setNextTrack(0)
		} else {
			p.setNextTrack(p.nowPlayingIdx + 1)
		}
	}
}

func (p *PlaybackManager) setTrack(idx int, next bool) error {
	if urlP, ok := p.player.(player.URLPlayer); ok {
		url := ""
		if idx >= 0 {
			var err error
			url, err = p.sm.Server.GetStreamURL(p.playQueue[idx].ID, p.transcodeCfg.ForceRawFile)
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
			track = p.playQueue[idx]
		}
		if next {
			return trP.SetNextTrack(track)
		}
		return trP.PlayTrack(track)
	}
	panic("Unsupported player type")
}

func (p *PlaybackManager) setNextTrack(idx int) error {
	return p.setTrack(idx, true)
}

// call BEFORE updating p.nowPlayingIdx
func (p *PlaybackManager) checkScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	playDur := p.playTimeStopwatch.Elapsed()
	if playDur.Seconds() < 0.1 || p.curTrackTime < 0.1 {
		return
	}
	pcnt := playDur.Seconds() / p.curTrackTime * 100
	timeThresholdMet := p.scrobbleCfg.ThresholdTimeSeconds >= 0 &&
		playDur.Seconds() >= float64(p.scrobbleCfg.ThresholdTimeSeconds)

	track := p.playQueue[p.nowPlayingIdx]
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

func (p *PlaybackManager) sendNowPlayingScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	track := p.playQueue[p.nowPlayingIdx]
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
func (p *PlaybackManager) deepCopyTrackSlice(tracks []*mediaprovider.Track) []*mediaprovider.Track {
	newTracks := make([]*mediaprovider.Track, len(tracks))
	for i, tr := range tracks {
		copy := *tr
		newTracks[i] = &copy
	}
	return newTracks
}

func (p *PlaybackManager) invokeOnSongChangeCallbacks() {
	if p.callbacksDisabled {
		return
	}
	for _, cb := range p.onSongChange {
		cb(p.NowPlaying(), p.lastScrobbled)
	}
	p.lastScrobbled = nil
}

func (pm *PlaybackManager) invokeNoArgCallbacks(cbs []func()) {
	if pm.callbacksDisabled {
		return
	}
	for _, cb := range cbs {
		cb()
	}
}

func (p *PlaybackManager) startPollTimePos() {
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
				p.doUpdateTimePos()
			}
		}
	}()
}

func (p *PlaybackManager) stopPollTimePos() {
	if p.cancelPollPos != nil {
		p.cancelPollPos()
		p.cancelPollPos = nil
	}
}

func (p *PlaybackManager) doUpdateTimePos() {
	if p.callbacksDisabled {
		return
	}
	s := p.player.GetStatus()
	if s.TimePos > p.latestTrackPosition {
		p.latestTrackPosition = s.TimePos
	}
	for _, cb := range p.onPlayTimeUpdate {
		cb(s.TimePos, s.Duration)
	}
}
