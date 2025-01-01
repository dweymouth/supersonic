package backend

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
)

// A high-level MediaProvider-aware playback engine, serves as an
// intermediary between the frontend and various Player backends.
type PlaybackManager struct {
	engine   *playbackEngine
	cmdQueue *playbackCommandQueue
	cfg      *AppConfig

	lastPlayTime float64
}

func NewPlaybackManager(
	ctx context.Context,
	s *ServerManager,
	p player.BasePlayer,
	playbackCfg *PlaybackConfig,
	scrobbleCfg *ScrobbleConfig,
	transcodeCfg *TranscodingConfig,
	appCfg *AppConfig,
) *PlaybackManager {
	e := NewPlaybackEngine(ctx, s, p, playbackCfg, scrobbleCfg, transcodeCfg)
	q := NewCommandQueue()
	pm := &PlaybackManager{
		engine:   e,
		cmdQueue: q,
		cfg:      appCfg,
	}
	pm.workaroundWindowsPlaybackIssue()
	go pm.runCmdQueue(ctx)
	return pm
}

func (p *PlaybackManager) workaroundWindowsPlaybackIssue() {
	// See https://github.com/dweymouth/supersonic/issues/483
	// On Windows, MPV sometimes fails to start playback when switching to a track
	// with a different sample rate than the previous. If this is detected,
	// send a command to the MPV player to force restart playback.
	p.OnPlayTimeUpdate(func(curTime, _ float64, _ bool) {
		p.lastPlayTime = curTime
	})
	p.OnSongChange(func(mediaprovider.MediaItem, *mediaprovider.Track) {
		if p.NowPlayingIndex() != len(p.engine.playQueue) && p.PlayerStatus().State == player.Playing {
			p.lastPlayTime = 0
			go func() {
				time.Sleep(300 * time.Millisecond)
				if p.lastPlayTime == 0 {
					log.Println("Play stall detected!")
					p.cmdQueue.addCommand(playbackCommand{Type: cmdForceRestartPlayback})
				}
			}()
		}
	})
}

func (p *PlaybackManager) CurrentPlayer() player.BasePlayer {
	return p.engine.CurrentPlayer()
}

func (p *PlaybackManager) OnPlayerChange(cb func()) {
	p.engine.onPlayerChange = append(p.engine.onPlayerChange, cb)
}

func (p *PlaybackManager) IsSeeking() bool {
	return p.engine.IsSeeking()
}

// Should only be called before quitting.
// Disables playback state callbacks being sent
func (p *PlaybackManager) DisableCallbacks() {
	p.engine.callbacksDisabled = true
}

// Gets the now playing media item, if any.
func (p *PlaybackManager) NowPlaying() mediaprovider.MediaItem {
	return p.engine.NowPlaying()
}

func (p *PlaybackManager) NowPlayingIndex() int {
	return p.engine.NowPlayingIndex()
}

// Sets a callback that is notified whenever a new song begins playing.
func (p *PlaybackManager) OnSongChange(cb func(nowPlaying mediaprovider.MediaItem, justScrobbledIfAny *mediaprovider.Track)) {
	p.engine.onSongChange = append(p.engine.onSongChange, cb)
}

// Registers a callback that is notified whenever the play time should be updated.
func (p *PlaybackManager) OnPlayTimeUpdate(cb func(curTime float64, totalTime float64, seeked bool)) {
	p.engine.onPlayTimeUpdate = append(p.engine.onPlayTimeUpdate, cb)
}

// Registers a callback that is notified whenever the loop mode changes.
func (p *PlaybackManager) OnLoopModeChange(cb func(LoopMode)) {
	p.engine.onLoopModeChange = append(p.engine.onLoopModeChange, cb)
}

// Registers a callback that is notified whenever the volume changes.
func (p *PlaybackManager) OnVolumeChange(cb func(int)) {
	p.engine.onVolumeChange = append(p.engine.onVolumeChange, cb)
}

// Registers a callback that is notified whenever the play queue changes.
func (p *PlaybackManager) OnQueueChange(cb func()) {
	p.engine.onQueueChange = append(p.engine.onQueueChange, cb)
}

// Registers a callback that is notified whenever the player has been seeked.
func (p *PlaybackManager) OnSeek(cb func()) {
	p.engine.onSeek = append(p.engine.onSeek, cb)
}

// Registers a callback that is notified whenever the player has been paused.
func (p *PlaybackManager) OnPaused(cb func()) {
	p.engine.onPaused = append(p.engine.onPaused, cb)
}

// Registers a callback that is notified whenever the player is stopped.
func (p *PlaybackManager) OnStopped(cb func()) {
	p.engine.onStopped = append(p.engine.onStopped, cb)
}

// Registers a callback that is notified whenever the player begins playing.
func (p *PlaybackManager) OnPlaying(cb func()) {
	p.engine.onPlaying = append(p.engine.onPlaying, cb)
}

// Loads the specified album into the play queue.
func (p *PlaybackManager) LoadAlbum(albumID string, insertQueueMode InsertQueueMode, shuffle bool) error {
	album, err := p.engine.sm.Server.GetAlbum(albumID)
	if err != nil {
		return err
	}
	p.LoadTracks(album.Tracks, insertQueueMode, shuffle)
	return nil
}

// Loads the specified playlist into the play queue.
func (p *PlaybackManager) LoadPlaylist(playlistID string, insertQueueMode InsertQueueMode, shuffle bool) error {
	playlist, err := p.engine.sm.Server.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	p.LoadTracks(playlist.Tracks, insertQueueMode, shuffle)
	return nil
}

// Load tracks into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *PlaybackManager) LoadTracks(tracks []*mediaprovider.Track, insertQueueMode InsertQueueMode, shuffle bool) {
	items := copyTrackSliceToMediaItemSlice(tracks)
	p.cmdQueue.LoadItems(items, insertQueueMode, shuffle)
}

// Load items into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *PlaybackManager) LoadItems(items []mediaprovider.MediaItem, insertQueueMode InsertQueueMode, shuffle bool) {
	p.cmdQueue.LoadItems(items, insertQueueMode, shuffle)
}

// Replaces the play queue with the given set of tracks.
// Does not stop playback if the currently playing track is in the new queue,
// but updates the now playing index to point to the first instance of the track in the new queue.
func (p *PlaybackManager) UpdatePlayQueue(items []mediaprovider.MediaItem) {
	p.cmdQueue.UpdatePlayQueue(items)
}

func (p *PlaybackManager) PlayAlbum(albumID string, firstTrack int, shuffle bool) error {
	if err := p.LoadAlbum(albumID, Replace, shuffle); err != nil {
		return err
	}
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainAlbum)
	}
	p.PlayTrackAt(firstTrack)
	return nil
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int, shuffle bool) error {
	if err := p.LoadPlaylist(playlistID, Replace, shuffle); err != nil {
		return err
	}
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	p.PlayTrackAt(firstTrack)
	return nil
}

func (p *PlaybackManager) PlayTrack(trackID string) error {
	tr, err := p.engine.sm.Server.GetTrack(trackID)
	if err != nil {
		return err
	}
	p.LoadTracks([]*mediaprovider.Track{tr}, Replace, false)
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	p.PlayFromBeginning()
	return nil
}

func (p *PlaybackManager) ShuffleArtistAlbums(artistID string) {
	artist, err := p.engine.sm.Server.GetArtist(artistID)
	if err != nil {
		log.Printf("failed to get artist: %v\n", err)
		return
	}
	if len(artist.Albums) == 0 {
		return
	}

	rand.Shuffle(len(artist.Albums), func(i, j int) {
		artist.Albums[i], artist.Albums[j] = artist.Albums[j], artist.Albums[i]
	})
	p.StopAndClearPlayQueue()
	for _, al := range artist.Albums {
		p.LoadAlbum(al.ID, Append, false)
	}

	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainAlbum)
	}
	p.PlayFromBeginning()
}

func (p *PlaybackManager) PlayArtistDiscography(artistID string, shuffleTracks bool) {
	tr, err := p.engine.sm.Server.GetArtistTracks(artistID)
	if err != nil {
		log.Printf("failed to get artist tracks: %v\n", err)
		return
	}
	p.LoadTracks(tr, Replace, shuffleTracks)
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		if shuffleTracks {
			p.SetReplayGainMode(player.ReplayGainTrack)
		} else {
			p.SetReplayGainMode(player.ReplayGainAlbum)
		}
	}
	p.PlayFromBeginning()
}

func (p *PlaybackManager) PlayFromBeginning() {
	p.cmdQueue.PlayTrackAt(0)
}

func (p *PlaybackManager) PlayTrackAt(idx int) {
	p.cmdQueue.PlayTrackAt(idx)
}

func (p *PlaybackManager) PlayRandomSongs(genreName string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.engine.sm.Server.GetRandomTracks(genreName, p.cfg.EnqueueBatchSize)
	})
}

func (p *PlaybackManager) PlaySimilarSongs(id string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.engine.sm.Server.GetSimilarTracks(id, p.cfg.EnqueueBatchSize)
	})
}

func (p *PlaybackManager) LoadRadioStation(station *mediaprovider.RadioStation, queueMode InsertQueueMode) {
	p.cmdQueue.LoadRadioStation(station, queueMode)
}

func (p *PlaybackManager) PlayRadioStation(station *mediaprovider.RadioStation) {
	p.LoadRadioStation(station, Replace)
	p.PlayFromBeginning()
}

func (p *PlaybackManager) fetchAndPlayTracks(fetchFn func() ([]*mediaprovider.Track, error)) {
	if songs, err := fetchFn(); err != nil {
		log.Printf("error fetching tracks: %s", err.Error())
	} else {
		p.LoadTracks(songs, Replace, false)
		if p.engine.replayGainCfg.Mode == ReplayGainAuto {
			p.SetReplayGainMode(player.ReplayGainTrack)
		}
		p.PlayFromBeginning()
	}
}

func (p *PlaybackManager) GetPlayQueue() []mediaprovider.MediaItem {
	return p.engine.GetPlayQueue()
}

// Any time the user changes the favorite status of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *PlaybackManager) OnTrackFavoriteStatusChanged(id string, fav bool) {
	p.engine.OnTrackFavoriteStatusChanged(id, fav)
}

// Any time the user changes the rating of a track elsewhere in the app,
// this should be called to ensure the in-memory track model is updated.
func (p *PlaybackManager) OnTrackRatingChanged(id string, rating int) {
	p.engine.OnTrackRatingChanged(id, rating)
}

func (p *PlaybackManager) RemoveTracksFromQueue(idxs []int) {
	p.cmdQueue.RemoveItemsFromQueue(idxs)
}

// Stop playback and clear the play queue.
func (p *PlaybackManager) StopAndClearPlayQueue() {
	p.cmdQueue.StopAndClearPlayQueue()
}

func (p *PlaybackManager) SetReplayGainOptions(config ReplayGainConfig) {
	p.engine.SetReplayGainOptions(config)
}

func (p *PlaybackManager) SetReplayGainMode(mode player.ReplayGainMode) {
	p.engine.SetReplayGainMode(mode)
}

// Changes the loop mode of the player to the next one.
// Useful for toggling UI elements, to change modes without knowing the current player mode.
func (p *PlaybackManager) SetNextLoopMode() {
	switch p.engine.loopMode {
	case LoopNone:
		p.cmdQueue.SetLoopMode(LoopAll)
	case LoopAll:
		p.cmdQueue.SetLoopMode(LoopOne)
	case LoopOne:
		p.cmdQueue.SetLoopMode(LoopNone)
	}
}

func (p *PlaybackManager) SetLoopMode(loopMode LoopMode) {
	p.cmdQueue.SetLoopMode(loopMode)
}

func (p *PlaybackManager) GetLoopMode() LoopMode {
	return p.engine.loopMode
}

func (p *PlaybackManager) PlayerStatus() player.Status {
	return p.engine.PlayerStatus()
}

func (p *PlaybackManager) SetVolume(vol int) {
	p.cmdQueue.SetVolume(vol)
}

func (p *PlaybackManager) Volume() int {
	return p.engine.CurrentPlayer().GetVolume()
}

func (p *PlaybackManager) SeekNext() {
	p.cmdQueue.SeekNext()
}

func (p *PlaybackManager) SeekBackOrPrevious() {
	p.cmdQueue.SeekBackOrPrevious()
}

// Seek to given absolute position in the current track by seconds.
func (p *PlaybackManager) SeekSeconds(sec float64) {
	p.cmdQueue.SeekSeconds(sec)
}

// Seek by given relative position in the current track by seconds.
func (p *PlaybackManager) SeekBySeconds(sec float64) {
	status := p.engine.PlayerStatus()
	target := status.TimePos + sec
	if target < 0 {
		target = 0
	} else if target > status.Duration {
		target = status.Duration
	}
	p.cmdQueue.SeekSeconds(target)
}

// Seek to a fractional position in the current track [0..1]
func (p *PlaybackManager) SeekFraction(fraction float64) {
	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}
	target := p.engine.curTrackDuration * fraction
	p.cmdQueue.SeekSeconds(target)
}

func (p *PlaybackManager) Stop() {
	p.cmdQueue.Stop()
}

func (p *PlaybackManager) Pause() {
	p.cmdQueue.Pause()
}

func (p *PlaybackManager) Continue() {
	p.cmdQueue.Continue()
}

func (p *PlaybackManager) PlayPause() {
	switch p.engine.PlayerStatus().State {
	case player.Playing:
		p.Pause()
	case player.Paused:
		p.Continue()
	case player.Stopped:
		p.PlayTrackAt(0)
	}
}

func (p *PlaybackManager) runCmdQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-p.cmdQueue.C():
			switch c.Type {
			case cmdStop:
				p.engine.Stop()
			case cmdContinue:
				p.engine.Continue()
			case cmdPause:
				p.engine.Pause()
			case cmdPlayTrackAt:
				p.engine.PlayTrackAt(c.Arg.(int))
			case cmdSeekSeconds:
				p.engine.SeekSeconds(c.Arg.(float64))
			case cmdSeekFwdBackN:
				p.engine.SeekFwdBackN(c.Arg.(int))
			case cmdVolume:
				p.engine.SetVolume(c.Arg.(int))
			case cmdLoopMode:
				p.engine.SetLoopMode(c.Arg.(LoopMode))
			case cmdStopAndClearPlayQueue:
				p.engine.StopAndClearPlayQueue()
			case cmdUpdatePlayQueue:
				p.engine.UpdatePlayQueue(c.Arg.([]mediaprovider.MediaItem))
			case cmdRemoveTracksFromQueue:
				p.engine.RemoveTracksFromQueue(c.Arg.([]int))
			case cmdLoadItems:
				p.engine.LoadItems(
					c.Arg.([]mediaprovider.MediaItem),
					c.Arg2.(InsertQueueMode),
					c.Arg3.(bool),
				)
			case cmdLoadRadioStation:
				p.engine.LoadRadioStation(
					c.Arg.(*mediaprovider.RadioStation),
					c.Arg2.(InsertQueueMode),
				)
			case cmdForceRestartPlayback:
				if mpv, ok := p.engine.CurrentPlayer().(*mpv.Player); ok {
					log.Println("Force-restarting MPV playback")
					mpv.ForceRestartPlayback()
				}
			}
		}
	}
}
