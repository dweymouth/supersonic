package backend

import (
	"context"
	"errors"
	"log"
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

// A high-level Subsonic-aware playback backend.
// Manages loading tracks into the Player queue,
// sending callbacks on play time updates and track changes.
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

	// to pass to onSongChange listeners; clear once listeners have been called
	lastScrobbled *mediaprovider.Track
	scrobbleCfg   *ScrobbleConfig
	transcodeCfg  *TranscodingConfig
	replayGainCfg ReplayGainConfig

	// registered callbacks
	onSongChange     []func(nowPlaying, justScrobbledIfAny *mediaprovider.Track)
	onPlayTimeUpdate []func(float64, float64)
	onLoopModeChange []func(player.LoopMode)
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
		ctx:          ctx,
		sm:           s,
		player:       p,
		scrobbleCfg:  scrobbleCfg,
		transcodeCfg: transcodeCfg,
	}
	p.OnTrackChange(func(tracknum int) {
		if tracknum >= len(pm.playQueue) {
			return
		}
		pm.checkScrobble() // scrobble the previous song if needed
		if pm.player.GetStatus().State == player.Playing {
			pm.playTimeStopwatch.Start()
		}
		pm.nowPlayingIdx = tracknum
		pm.curTrackTime = float64(pm.playQueue[pm.nowPlayingIdx].Duration)
		pm.sendNowPlayingScrobble() // Must come before invokeOnChangeCallbacks b/c track may immediately be scrobbled
		pm.invokeOnSongChangeCallbacks()
		pm.doUpdateTimePos()
	})
	p.OnSeek(func() {
		pm.doUpdateTimePos()
		pm.invokeNoArgCallbacks(pm.onSeek)
	})
	p.OnStopped(func() {
		pm.playTimeStopwatch.Stop()
		pm.checkScrobble()
		pm.stopPollTimePos()
		pm.doUpdateTimePos()
		pm.invokeOnSongChangeCallbacks()
		pm.invokeNoArgCallbacks(pm.onStopped)
	})
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
func (p *PlaybackManager) OnLoopModeChange(cb func(player.LoopMode)) {
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

func (p *PlaybackManager) LoadTracks(tracks []*mediaprovider.Track, appendToQueue, shuffle bool) error {
	if !appendToQueue {
		p.player.Stop()
		p.nowPlayingIdx = 0
		p.playQueue = nil
	}
	nums := util.Range(len(tracks))
	if shuffle {
		util.ShuffleSlice(nums)
	}
	for _, i := range nums {

		if urlP, ok := p.player.(player.URLPlayer); ok {
			url, err := p.sm.Server.GetStreamURL(tracks[i].ID, p.transcodeCfg.ForceRawFile)
			if err != nil {
				return err
			}
			urlP.AppendFile(url)
		} else if trP, ok := p.player.(player.TrackPlayer); ok {
			trP.AppendTrack(tracks[i])
		} else {
			panic("unsupported player type")
		}
		// ensure a deep copy of the track info so that we can maintain our own state
		// (tracking play count increases, favorite, and rating) without messing up
		// other views' track models
		tr := *tracks[i]
		p.playQueue = append(p.playQueue, &tr)
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
	return p.player.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int, shuffle bool) error {
	if err := p.LoadPlaylist(playlistID, false, shuffle); err != nil {
		return err
	}
	if p.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	return p.player.PlayTrackAt(firstTrack)
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
	return p.player.PlayTrackAt(0)
}

func (p *PlaybackManager) PlayTrackAt(idx int) error {
	return p.player.PlayTrackAt(idx)
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
	pq := make([]*mediaprovider.Track, len(p.playQueue))
	for i, tr := range p.playQueue {
		copy := *tr
		pq[i] = &copy
	}
	return pq
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
	rmCount := 0
	idSet := sharedutil.ToSet(trackIDs)
	isPlayingTrackRemoved := false
	for i, tr := range p.playQueue {
		if _, ok := idSet[tr.ID]; ok {
			// removing this track
			if i == p.NowPlayingIndex() {
				isPlayingTrackRemoved = true
				// If we are removing the currently playing track, we need to scrobble it
				p.checkScrobble()
			}
			if err := p.player.RemoveTrackAt(i - rmCount); err == nil {
				rmCount++
			} else {
				log.Printf("error removing track: %v", err.Error())
				// did not remove this track
				newQueue = append(newQueue, tr)
			}
		} else {
			// not removing this track
			newQueue = append(newQueue, tr)
		}
	}
	p.playQueue = newQueue
	p.nowPlayingIdx = p.player.GetStatus().PlaylistPos
	// fire on song change callbacks in case the playing track was removed
	if isPlayingTrackRemoved {
		p.invokeOnSongChangeCallbacks()
	}
}

// Stop playback and clear the play queue.
func (p *PlaybackManager) StopAndClearPlayQueue() {
	p.player.Stop()
	p.player.ClearPlayQueue()
	p.doUpdateTimePos()
	p.playQueue = nil
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

	if config.Mode == ReplayGainAuto {
		mode = player.ReplayGainTrack
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
func (p *PlaybackManager) SetNextLoopMode() error {
	var err error
	switch p.GetLoopMode() {
	case player.LoopNone:
		err = p.SetLoopMode(player.LoopAll)
	case player.LoopAll:
		err = p.SetLoopMode(player.LoopOne)
	case player.LoopOne:
		err = p.SetLoopMode(player.LoopNone)
	default:
		return nil
	}

	if err != nil {
		return err
	}
	for _, cb := range p.onLoopModeChange {
		cb(p.player.GetLoopMode())
	}

	return nil
}

func (p *PlaybackManager) SetLoopMode(loopMode player.LoopMode) error {
	if err := p.player.SetLoopMode(player.LoopMode(loopMode)); err != nil {
		return err
	}

	for _, cb := range p.onLoopModeChange {
		cb(loopMode)
	}

	return nil
}

func (p *PlaybackManager) GetLoopMode() player.LoopMode {
	return p.player.GetLoopMode()
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
	return p.player.SeekNext()
}

func (p *PlaybackManager) SeekBackOrPrevious() error {
	if p.player.GetStatus().TimePos > 3 {
		return p.player.SeekSeconds(0)
	}
	return p.player.SeekPrevious()
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
