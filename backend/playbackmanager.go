package backend

import (
	"context"
	"errors"
	"log"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
)

// A high-level MediaProvider-aware playback engine, serves as an
// intermediary between the frontend and various Player backends.
type PlaybackManager struct {
	engine *playbackEngine
}

type MediaType int

const (
	MediaTypeTrack MediaType = iota
	MediaTypeRadioStation
)

type NowPlayingMetadata struct {
	Type       MediaType
	ID         string
	Title      string
	Artists    []string
	ArtistIDs  []string
	Album      string
	AlbumID    string
	CoverArtID string
	Duration   int
}

func NewPlaybackManager(
	ctx context.Context,
	s *ServerManager,
	p player.BasePlayer,
	scrobbleCfg *ScrobbleConfig,
	transcodeCfg *TranscodingConfig,
) *PlaybackManager {
	return &PlaybackManager{
		engine: NewPlaybackEngine(ctx, s, p, scrobbleCfg, transcodeCfg),
	}
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

// Gets the curently playing song, if any.
func (p *PlaybackManager) NowPlaying() *NowPlayingMetadata {
	return p.engine.NowPlaying()
}

func (p *PlaybackManager) NowPlayingMediaItem() mediaprovider.MediaItem {
	return p.engine.NowPlayingMediaItem()
}

func (p *PlaybackManager) NowPlayingIndex() int {
	return p.engine.NowPlayingIndex()
}

// Sets a callback that is notified whenever a new song begins playing.
func (p *PlaybackManager) OnSongChange(cb func(nowPlaying *NowPlayingMetadata, justScrobbledIfAny *mediaprovider.Track)) {
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
	return p.LoadTracks(album.Tracks, insertQueueMode, shuffle)
}

// Loads the specified playlist into the play queue.
func (p *PlaybackManager) LoadPlaylist(playlistID string, insertQueueMode InsertQueueMode, shuffle bool) error {
	playlist, err := p.engine.sm.Server.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	return p.LoadTracks(playlist.Tracks, insertQueueMode, shuffle)
}

// Load tracks into the play queue.
// If replacing the current queue (!appendToQueue), playback will be stopped.
func (p *PlaybackManager) LoadTracks(tracks []*mediaprovider.Track, insertQueueMode InsertQueueMode, shuffle bool) error {
	return p.engine.LoadTracks(tracks, insertQueueMode, shuffle)
}

// Replaces the play queue with the given set of tracks.
// Does not stop playback if the currently playing track is in the new queue,
// but updates the now playing index to point to the first instance of the track in the new queue.
func (p *PlaybackManager) UpdatePlayQueue(tracks []*mediaprovider.Track) error {
	return p.engine.UpdatePlayQueue(tracks)
}

func (p *PlaybackManager) PlayAlbum(albumID string, firstTrack int, shuffle bool) error {
	if err := p.LoadAlbum(albumID, Replace, shuffle); err != nil {
		return err
	}
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainAlbum)
	}
	return p.PlayTrackAt(firstTrack)
}

func (p *PlaybackManager) PlayPlaylist(playlistID string, firstTrack int, shuffle bool) error {
	if err := p.LoadPlaylist(playlistID, Replace, shuffle); err != nil {
		return err
	}
	if p.engine.replayGainCfg.Mode == ReplayGainAuto {
		p.SetReplayGainMode(player.ReplayGainTrack)
	}
	return p.PlayTrackAt(firstTrack)
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
	return p.PlayFromBeginning()
}

func (p *PlaybackManager) PlayFromBeginning() error {
	return p.engine.PlayTrackAt(0)
}

func (p *PlaybackManager) PlayTrackAt(idx int) error {
	return p.engine.PlayTrackAt(idx)
}

func (p *PlaybackManager) PlayRandomSongs(genreName string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.engine.sm.Server.GetRandomTracks(genreName, 100)
	})
}

func (p *PlaybackManager) PlaySimilarSongs(id string) {
	p.fetchAndPlayTracks(func() ([]*mediaprovider.Track, error) {
		return p.engine.sm.Server.GetSimilarTracks(id, 100)
	})
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

func (p *PlaybackManager) RemoveTracksFromQueue(trackIDs []string) {
	p.engine.RemoveTracksFromQueue(trackIDs)
}

// Stop playback and clear the play queue.
func (p *PlaybackManager) StopAndClearPlayQueue() {
	p.engine.StopAndClearPlayQueue()
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
		p.engine.SetLoopMode(LoopAll)
	case LoopAll:
		p.engine.SetLoopMode(LoopOne)
	case LoopOne:
		p.engine.SetLoopMode(LoopNone)

	}
}

func (p *PlaybackManager) SetLoopMode(loopMode LoopMode) {
	p.engine.SetLoopMode(loopMode)
}

func (p *PlaybackManager) GetLoopMode() LoopMode {
	return p.engine.loopMode
}

func (p *PlaybackManager) PlayerStatus() player.Status {
	return p.engine.PlayerStatus()
}

func (p *PlaybackManager) SetVolume(vol int) error {
	return p.engine.SetVolume(vol)
}

func (p *PlaybackManager) Volume() int {
	return p.engine.CurrentPlayer().GetVolume()
}

func (p *PlaybackManager) SeekNext() error {
	return p.engine.SeekNext()
}

func (p *PlaybackManager) SeekBackOrPrevious() error {
	return p.engine.SeekBackOrPrevious()
}

// Seek to given absolute position in the current track by seconds.
func (p *PlaybackManager) SeekSeconds(sec float64) error {
	return p.engine.SeekSeconds(sec)
}

// Seek to a fractional position in the current track [0..1]
func (p *PlaybackManager) SeekFraction(fraction float64) error {
	if fraction < 0 {
		fraction = 0
	} else if fraction > 1 {
		fraction = 1
	}
	target := p.engine.curTrackTime * fraction
	return p.engine.SeekSeconds(target)
}

func (p *PlaybackManager) Stop() error {
	return p.engine.Stop()
}

func (p *PlaybackManager) Pause() error {
	return p.engine.Pause()
}

func (p *PlaybackManager) Continue() error {
	return p.engine.Continue()
}

func (p *PlaybackManager) PlayPause() error {
	switch p.engine.PlayerStatus().State {
	case player.Playing:
		return p.engine.Pause()
	case player.Paused:
		return p.engine.Continue()
	case player.Stopped:
		return p.engine.PlayTrackAt(0)
	}
	return errors.New("unreached - invalid player state")
}
