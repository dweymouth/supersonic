package backend

import (
	"context"
	"log"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/sharedutil"
)

const (
	ReplayGainNone  = string(player.ReplayGainNone)
	ReplayGainAlbum = string(player.ReplayGainAlbum)
	ReplayGainTrack = string(player.ReplayGainTrack)
)

type LoopMode int

const (
	LoopModeNone LoopMode = LoopMode(player.LoopNone)
	LoopModeAll  LoopMode = LoopMode(player.LoopAll)
	LoopModeOne  LoopMode = LoopMode(player.LoopOne)
)

// A high-level Subsonic-aware playback backend.
// Manages loading tracks into the Player queue,
// sending callbacks on play time updates and track changes.
type PlaybackManager struct {
	ctx           context.Context
	cancelPollPos context.CancelFunc
	pollingTick   *time.Ticker
	sm            *ServerManager
	player        *player.Player

	playTimeStopwatch util.Stopwatch
	curTrackTime      float64
	callbacksDisabled bool

	playQueue     []*mediaprovider.Track
	nowPlayingIdx int64

	// to pass to onSongChange listeners; clear once listeners have been called
	lastScrobbled *mediaprovider.Track
	scrobbleCfg   *ScrobbleConfig

	onSongChange     []func(nowPlaying, justScrobbledIfAny *mediaprovider.Track)
	onPlayTimeUpdate []func(float64, float64)
	onLoopModeChange []func(LoopMode)
	onVolumeChange   []func(int)
}

func NewPlaybackManager(
	ctx context.Context,
	s *ServerManager,
	p *player.Player,
	scrobbleCfg *ScrobbleConfig,
) *PlaybackManager {
	// clamp to 99% to avoid any possible rounding issues
	scrobbleCfg.ThresholdPercent = clamp(scrobbleCfg.ThresholdPercent, 0, 99)
	pm := &PlaybackManager{
		ctx:         ctx,
		sm:          s,
		player:      p,
		scrobbleCfg: scrobbleCfg,
	}
	p.OnTrackChange(func(tracknum int64) {
		if tracknum >= int64(len(pm.playQueue)) {
			return
		}
		pm.checkScrobble()
		if pm.player.GetStatus().State == player.Playing {
			pm.playTimeStopwatch.Start()
		}
		pm.nowPlayingIdx = tracknum
		pm.curTrackTime = float64(pm.playQueue[pm.nowPlayingIdx].Duration)
		pm.invokeOnSongChangeCallbacks()
		pm.doUpdateTimePos()
		pm.sendNowPlayingScrobble()
	})
	p.OnSeek(func() {
		pm.doUpdateTimePos()
	})
	p.OnStopped(func() {
		pm.playTimeStopwatch.Stop()
		pm.checkScrobble()
		pm.stopPollTimePos()
		pm.doUpdateTimePos()
		pm.invokeOnSongChangeCallbacks()
	})
	p.OnPaused(func() {
		pm.playTimeStopwatch.Stop()
		pm.stopPollTimePos()
	})
	p.OnPlaying(func() {
		pm.playTimeStopwatch.Start()
		pm.startPollTimePos()
	})

	s.OnLogout(func() {
		pm.StopAndClearPlayQueue()
	})

	return pm
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
	if len(p.playQueue) == 0 || p.player.GetStatus().State == player.Stopped {
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
		url, err := p.sm.Server.GetStreamURL(tracks[i].ID)
		if err != nil {
			return err
		}
		p.player.AppendFile(url)
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
	if firstTrack <= 0 {
		return p.player.PlayFromBeginning()
	}
	return p.player.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int, shuffle bool) error {
	if err := p.LoadPlaylist(playlistID, false, shuffle); err != nil {
		return err
	}
	if firstTrack <= 0 {
		return p.player.PlayFromBeginning()
	}
	return p.player.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayFromBeginning() error {
	return p.player.PlayFromBeginning()
}

func (p *PlaybackManager) PlayTrackAt(idx int) error {
	return p.player.PlayTrackAt(idx)
}

func (p *PlaybackManager) PlayRandomSongs(genreName string) {
	if songs, err := p.sm.Server.GetRandomTracks(genreName, 100); err != nil {
		log.Printf("error getting random songs: %s", err.Error())
	} else {
		p.LoadTracks(songs, false, false)
		p.PlayFromBeginning()
	}
}

func (p *PlaybackManager) PlaySimilarSongs(id string) {
	if songs, err := p.sm.Server.GetSimilarTracks(id, 100); err != nil {
		log.Printf("error getting similar songs: %s", err.Error())
	} else {
		p.LoadTracks(songs, false, false)
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
	p.player.SetReplayGainOptions(player.ReplayGainOptions{
		Mode:            player.ReplayGainMode(config.Mode),
		PreventClipping: config.PreventClipping,
		PreampGain:      config.PreampGainDB,
	})
}

// Changes the loop mode of the player to the next one.
// Useful for toggling UI elements, to change modes without knowing the current player mode.
func (p *PlaybackManager) SetNextLoopMode() error {
	if err := p.player.SetNextLoopMode(); err != nil {
		return err
	}

	for _, cb := range p.onLoopModeChange {
		cb(LoopMode(p.player.GetLoopMode()))
	}

	return nil
}

func (p *PlaybackManager) SetLoopMode(loopMode LoopMode) error {
	if err := p.player.SetLoopMode(player.LoopMode(loopMode)); err != nil {
		return err
	}

	for _, cb := range p.onLoopModeChange {
		cb(loopMode)
	}

	return nil
}

func (p *PlaybackManager) LoopMode() LoopMode {
	return LoopMode(p.player.GetLoopMode())
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
	if timeThresholdMet || pcnt >= float64(p.scrobbleCfg.ThresholdPercent) {
		song := p.playQueue[p.nowPlayingIdx]
		log.Printf("Scrobbling %q", song.Name)
		song.PlayCount += 1
		p.lastScrobbled = song
		go p.sm.Server.Scrobble(song.ID, true)
	}
	p.playTimeStopwatch.Reset()
}

func (p *PlaybackManager) sendNowPlayingScrobble() {
	if !p.scrobbleCfg.Enabled || len(p.playQueue) == 0 || p.nowPlayingIdx < 0 {
		return
	}
	song := p.playQueue[p.nowPlayingIdx]
	go p.sm.Server.Scrobble(song.ID, false)
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

func (p *PlaybackManager) startPollTimePos() {
	ctx, cancel := context.WithCancel(p.ctx)
	p.cancelPollPos = cancel
	p.pollingTick = time.NewTicker(250 * time.Millisecond)

	// TODO: fix occasional nil pointer dereference on app quit
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.pollingTick.Stop()
				p.pollingTick = nil
				return
			case <-p.pollingTick.C:
				p.doUpdateTimePos()
			}
		}
	}()
}

func (p *PlaybackManager) doUpdateTimePos() {
	if p.callbacksDisabled {
		return
	}
	s := p.player.GetStatus()
	for _, cb := range p.onPlayTimeUpdate {
		cb(s.TimePos, s.Duration)
	}
}

func (p *PlaybackManager) stopPollTimePos() {
	if p.cancelPollPos != nil {
		p.cancelPollPos()
		p.cancelPollPos = nil
	}
	if p.pollingTick != nil {
		p.pollingTick.Stop()
	}
}
