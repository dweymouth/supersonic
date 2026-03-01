package jukebox

import (
	"context"
	"log"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
)

const (
	stopped = 0
	playing = 1
	paused  = 2
)

type JukeboxPlayer struct {
	player.BasePlayerCallbackImpl

	provider mediaprovider.JukeboxProvider

	state   int // stopped, playing, paused
	volume  int
	seeking bool

	curTrack           int
	queueLength        int
	curTrackDuration   float64
	startTrackTime     float64
	startedAtUnixMilli int64

	// Queue management - tracks loaded via LoadQueue
	queueLoaded bool
	queueTracks []*mediaprovider.Track

	// For status polling
	ctx       context.Context
	cancel    context.CancelFunc
	lastState int

	// Cached audio info from last status update
	cachedAudioInfo MediaInfo
}

// MediaInfo contains audio information for the currently playing track.
type MediaInfo struct {
	Samplerate   int    // Hz
	ChannelCount int    // number of channels
	Codec        string // codec name (e.g., "flac", "mp3")
	Bitrate      int    // bits per second
}

// NewJukeboxPlayer creates a new JukeboxPlayer for the given JukeboxProvider.
func NewJukeboxPlayer(ctx context.Context, provider mediaprovider.JukeboxProvider) *JukeboxPlayer {
	playerCtx, cancel := context.WithCancel(ctx)
	j := &JukeboxPlayer{
		provider:  provider,
		ctx:       playerCtx,
		cancel:    cancel,
		lastState: stopped,
	}

	// Get initial status after a brief delay to avoid disrupting other clients
	go func() {
		time.Sleep(500 * time.Millisecond)
		// Do an initial sync which will invoke callbacks if MPD is already playing
		j.syncStatus()

		// Try to use event-driven updates if available, otherwise fall back to polling
		if watcher, ok := provider.(mediaprovider.JukeboxWatcher); ok {
			if events, err := watcher.WatchPlaybackEvents(playerCtx); err == nil {
				log.Println("Using MPD idle events for status updates")
				j.watchEvents(events)
				return
			}
			log.Println("Failed to start event watcher, falling back to polling")
		}
		j.pollStatus()
	}()

	return j
}

func (j *JukeboxPlayer) pollStatus() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-j.ctx.Done():
			return
		case <-ticker.C:
			j.syncStatus()
		}
	}
}

// watchEvents uses event-driven updates instead of polling
// Also uses a ticker to periodically update audio info during playback
func (j *JukeboxPlayer) watchEvents(events <-chan string) {
	// Ticker for periodic audio info updates during playback
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-j.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				// Channel closed, fall back to polling
				log.Println("Event channel closed, falling back to polling")
				j.pollStatus()
				return
			}
			// Sync status on any relevant event
			if event == "player" || event == "mixer" || event == "options" {
				j.syncStatus()
			}
		case <-ticker.C:
			// Periodic sync for audio info updates during playback
			if j.state == playing {
				j.syncStatus()
			}
		}
	}
}

func (j *JukeboxPlayer) syncStatus() {
	status, err := j.provider.JukeboxGetStatus()
	if err != nil {
		log.Printf("Error getting jukebox status: %v", err)
		return
	}

	// Update volume if changed externally
	if status.Volume != j.volume {
		j.volume = status.Volume
		j.InvokeOnVolumeChange(j.volume)
	}

	// Update cached audio info and notify if changed
	newAudioInfo := MediaInfo{
		Samplerate:   status.SampleRate,
		ChannelCount: status.Channels,
		Codec:        status.Codec,
		Bitrate:      status.Bitrate * 1000, // Convert kbps to bps
	}
	if newAudioInfo != j.cachedAudioInfo {
		j.cachedAudioInfo = newAudioInfo
		j.InvokeOnAudioInfoChange()
	}

	// Detect state changes from external sources
	newState := stopped
	if status.Playing {
		newState = playing
	} else if status.CurrentTrack >= 0 && status.PositionSeconds > 0 {
		newState = paused
	}

	// Handle state transitions
	if newState != j.lastState {
		j.state = newState
		switch newState {
		case playing:
			j.InvokeOnPlaying()
		case paused:
			j.InvokeOnPaused()
		case stopped:
			j.InvokeOnStopped()
		}
		j.lastState = newState
	}

	// Track change detection
	// If MPD advances to the next track (currentTrack increased), invoke callback
	// so that scrobbling works for tracks in Supersonic's queue
	if status.CurrentTrack != j.curTrack && status.CurrentTrack >= 0 {
		oldTrack := j.curTrack
		j.curTrack = status.CurrentTrack
		// If this looks like a natural advance (next track in queue), invoke callback
		// This allows scrobbling to work when tracks finish playing
		if status.CurrentTrack == oldTrack+1 || (oldTrack > 0 && status.CurrentTrack == 0) {
			j.InvokeOnTrackChange()
		}
	}
}

func (j *JukeboxPlayer) SetVolume(vol int) error {
	if err := j.provider.JukeboxSetVolume(vol); err != nil {
		return err
	}
	j.volume = vol
	return nil
}

func (j *JukeboxPlayer) GetVolume() int {
	return j.volume
}

func (j *JukeboxPlayer) Continue() error {
	if j.state == playing {
		return nil
	}
	if err := j.startAndUpdateTime(); err != nil {
		return err
	}

	j.state = playing
	j.lastState = playing
	j.InvokeOnPlaying()
	return nil
}

func (j *JukeboxPlayer) Pause() error {
	if j.state != playing {
		return nil
	}
	if err := j.provider.JukeboxStop(); err != nil {
		return err
	}
	j.state = paused
	j.lastState = paused
	j.InvokeOnPaused()
	return nil
}

func (j *JukeboxPlayer) Stop(force bool) error {
	if j.state == stopped {
		return nil
	}
	// Check if we're shutting down or force-stopping (e.g., switching providers)
	// In these cases, don't affect MPD state - other clients might be using it
	// or the user might switch back to MPD and expect the queue to still be there
	select {
	case <-j.ctx.Done():
		// Context cancelled (shutdown) - just update local state
		j.state = stopped
		j.lastState = stopped
		return nil
	default:
		if force {
			// Force stop (e.g., switching providers) - just update local state
			j.state = stopped
			j.lastState = stopped
			j.queueLoaded = false
			j.queueTracks = nil
			return nil
		}
	}
	// Normal stop - send commands to MPD
	if err := j.provider.JukeboxStop(); err != nil {
		return err
	}
	if err := j.provider.JukeboxClear(); err != nil {
		return err
	}
	// Clear the loaded queue state since we cleared the MPD queue
	j.queueLoaded = false
	j.queueTracks = nil
	j.queueLength = 0
	j.state = stopped
	j.lastState = stopped
	j.InvokeOnStopped()
	return nil
}

func (j *JukeboxPlayer) PlayTrack(track *mediaprovider.Track, _ float64) error {
	// JukeboxSet clears the MPD queue, so clear our loaded queue state
	j.queueLoaded = false
	j.queueTracks = nil

	if err := j.provider.JukeboxSet(track.ID); err != nil {
		return err
	}
	j.startTrackTime = 0
	if err := j.startAndUpdateTime(); err != nil {
		return err
	}

	j.curTrack = 0
	j.queueLength = 1
	j.curTrackDuration = track.Duration.Seconds()
	j.state = playing
	j.lastState = playing
	j.InvokeOnPlaying()
	j.InvokeOnTrackChange()
	return nil
}

func (j *JukeboxPlayer) SetNextTrack(track *mediaprovider.Track) error {
	if track == nil {
		return nil
	}
	// we need to replace the last track in the queue, remove it first
	if j.curTrack < j.queueLength-1 {
		if err := j.provider.JukeboxRemove(j.curTrack + 1); err != nil {
			return err
		}
		j.queueLength -= 1
	}
	// append the new track to the queue
	if err := j.provider.JukeboxAdd(track.ID); err != nil {
		return err
	}
	j.queueLength += 1
	return nil
}

func (j *JukeboxPlayer) SeekSeconds(secs float64) error {
	j.seeking = true
	err := j.provider.JukeboxSeek(j.curTrack, int(secs))
	j.seeking = false
	j.InvokeOnSeek()
	return err
}

func (j *JukeboxPlayer) IsSeeking() bool {
	return j.seeking
}

func (j *JukeboxPlayer) GetStatus() player.Status {
	// Get fresh status from server
	status, err := j.provider.JukeboxGetStatus()
	if err != nil {
		// Fall back to cached state
		state := player.Stopped
		if j.state == playing {
			state = player.Playing
		} else if j.state == paused {
			state = player.Paused
		}
		return player.Status{State: state}
	}

	state := player.Stopped
	if status.Playing {
		state = player.Playing
	} else if status.PositionSeconds > 0 {
		state = player.Paused
	}

	return player.Status{
		State:   state,
		TimePos: status.PositionSeconds,
	}
}

func (j *JukeboxPlayer) Destroy() {
	if j.cancel != nil {
		j.cancel()
	}
}

// GetMediaInfo returns the audio info for the currently playing track.
func (j *JukeboxPlayer) GetMediaInfo() (MediaInfo, error) {
	return j.cachedAudioInfo, nil
}

// LoadQueue loads all tracks into the jukebox queue.
// This implements the player.QueuePlayer interface.
func (j *JukeboxPlayer) LoadQueue(tracks []*mediaprovider.Track) error {
	// Clear the current queue
	if err := j.provider.JukeboxClear(); err != nil {
		return err
	}

	// Add all tracks to the queue
	for _, track := range tracks {
		if err := j.provider.JukeboxAdd(track.ID); err != nil {
			return err
		}
	}

	j.queueLoaded = true
	j.queueTracks = tracks
	j.queueLength = len(tracks)
	j.curTrack = -1 // Not playing yet

	return nil
}

// PlayQueueIndex starts playback at the specified index in the loaded queue.
// This implements the player.QueuePlayer interface.
func (j *JukeboxPlayer) PlayQueueIndex(idx int) error {
	if idx < 0 || idx >= j.queueLength {
		return nil
	}

	// Use JukeboxPlay to start at specific index
	if err := j.provider.JukeboxPlay(idx); err != nil {
		return err
	}

	j.curTrack = idx
	if idx < len(j.queueTracks) {
		j.curTrackDuration = j.queueTracks[idx].Duration.Seconds()
	}
	j.state = playing
	j.lastState = playing
	j.InvokeOnPlaying()
	j.InvokeOnTrackChange()

	return nil
}

// IsQueueLoaded returns true if a queue has been loaded via LoadQueue.
// This implements the player.QueuePlayer interface.
func (j *JukeboxPlayer) IsQueueLoaded() bool {
	return j.queueLoaded
}

// ClearLoadedQueue clears the loaded queue state.
// This implements the player.QueuePlayer interface.
func (j *JukeboxPlayer) ClearLoadedQueue() {
	j.queueLoaded = false
	j.queueTracks = nil
}

func (j *JukeboxPlayer) startAndUpdateTime() error {
	beforeStart := time.Now()
	if err := j.provider.JukeboxStart(); err != nil {
		return err
	}
	afterStart := time.Now()

	// assume track started playing at (ie has been playing for)
	// half the round-trip latency
	j.startedAtUnixMilli = time.Now().Add(-afterStart.Sub(beforeStart)).UnixMilli()
	return nil
}
