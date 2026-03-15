package jukebox

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
)

const (
	stopped = 0
	playing = 1
	paused  = 2
)

// JukeboxPlayer wraps a JukeboxProvider (e.g. MPD) so Supersonic's playback
// engine can drive it through the standard player.BasePlayer interface.
//
// Thread safety: all exported methods and syncStatus (called from a background
// goroutine) must hold mu before accessing any mutable fields.  Callbacks are
// always invoked AFTER releasing mu to avoid re-entrant deadlocks.
type JukeboxPlayer struct {
	player.BasePlayerCallbackImpl

	provider mediaprovider.JukeboxProvider

	mu sync.Mutex // protects all fields below

	state   int // stopped, playing, paused
	volume  int
	seeking bool

	curTrack           int
	queueLength        int
	curTrackDuration   float64
	startTrackTime     float64
	startedAtUnixMilli int64
	cachedTimePos      float64 // last known playback position (seconds)

	// Queue management - tracks loaded via LoadQueue
	queueLoaded bool
	queueTracks []*mediaprovider.Track

	// For background polling/event loop
	ctx    context.Context
	cancel context.CancelFunc

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
		provider: provider,
		ctx:      playerCtx,
		cancel:   cancel,
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

// watchEvents uses event-driven updates instead of polling.
// A secondary 500 ms ticker keeps the cached position fresh while playing.
func (j *JukeboxPlayer) watchEvents(events <-chan string) {
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
			if event == "player" || event == "mixer" || event == "options" {
				j.syncStatus()
			}
		case <-ticker.C:
			// Periodic sync for position updates while playing
			j.mu.Lock()
			isPlaying := j.state == playing
			j.mu.Unlock()
			if isPlaying {
				j.syncStatus()
			}
		}
	}
}

// syncStatus fetches fresh status from the jukebox server and updates cached
// state.  It is safe to call from any goroutine; callbacks are invoked after
// releasing the lock.
func (j *JukeboxPlayer) syncStatus() {
	status, err := j.provider.JukeboxGetStatus()
	if err != nil {
		log.Printf("Error getting jukebox status: %v", err)
		return
	}

	// Determine the new playback state.
	// Use CurrentTrack >= 0 (not PositionSeconds > 0) so "paused at start" is
	// correctly reported as Paused rather than Stopped.
	newState := stopped
	if status.Playing {
		newState = playing
	} else if status.CurrentTrack >= 0 {
		newState = paused
	}

	newAudioInfo := MediaInfo{
		Samplerate:   status.SampleRate,
		ChannelCount: status.Channels,
		Codec:        status.Codec,
		Bitrate:      status.Bitrate * 1000, // Convert kbps to bps
	}

	// Collect which callbacks to invoke, then release the lock before calling
	// them to prevent re-entrant deadlocks.
	var invokeVolChange bool
	var newVol int
	var invokeAudioInfo bool
	var invokeStateChange int // 0 = no change
	var invokeTrackChange bool

	j.mu.Lock()

	if status.Volume != j.volume {
		j.volume = status.Volume
		newVol = status.Volume
		invokeVolChange = true
	}

	if newAudioInfo != j.cachedAudioInfo {
		j.cachedAudioInfo = newAudioInfo
		invokeAudioInfo = true
	}

	j.cachedTimePos = status.PositionSeconds

	if newState != j.state {
		j.state = newState
		invokeStateChange = newState
	}

	// Track-change detection: natural queue advance or wrap-around.
	if status.CurrentTrack != j.curTrack && status.CurrentTrack >= 0 {
		oldTrack := j.curTrack
		j.curTrack = status.CurrentTrack
		if status.CurrentTrack == oldTrack+1 || (oldTrack > 0 && status.CurrentTrack == 0) {
			invokeTrackChange = true
		}
	}

	j.mu.Unlock()

	// Invoke callbacks outside the lock.
	if invokeVolChange {
		j.InvokeOnVolumeChange(newVol)
	}
	if invokeAudioInfo {
		j.InvokeOnAudioInfoChange()
	}
	switch invokeStateChange {
	case playing:
		j.InvokeOnPlaying()
	case paused:
		j.InvokeOnPaused()
	case stopped:
		j.InvokeOnStopped()
	}
	if invokeTrackChange {
		j.InvokeOnTrackChange()
	}
}

func (j *JukeboxPlayer) SetVolume(vol int) error {
	if err := j.provider.JukeboxSetVolume(vol); err != nil {
		return err
	}
	j.mu.Lock()
	j.volume = vol
	j.mu.Unlock()
	return nil
}

func (j *JukeboxPlayer) GetVolume() int {
	j.mu.Lock()
	v := j.volume
	j.mu.Unlock()
	return v
}

func (j *JukeboxPlayer) Continue() error {
	j.mu.Lock()
	if j.state == playing {
		j.mu.Unlock()
		return nil
	}
	j.mu.Unlock()

	if err := j.startAndUpdateTime(); err != nil {
		return err
	}

	j.mu.Lock()
	j.state = playing
	j.mu.Unlock()

	j.InvokeOnPlaying()
	return nil
}

func (j *JukeboxPlayer) Pause() error {
	j.mu.Lock()
	if j.state != playing {
		j.mu.Unlock()
		return nil
	}
	j.mu.Unlock()

	if err := j.provider.JukeboxStop(); err != nil {
		return err
	}

	j.mu.Lock()
	j.state = paused
	j.mu.Unlock()

	j.InvokeOnPaused()
	return nil
}

func (j *JukeboxPlayer) Stop(force bool) error {
	j.mu.Lock()
	if j.state == stopped {
		j.mu.Unlock()
		return nil
	}

	// Check for shutdown (context cancelled) or provider-switch (force).
	// In both cases only update local state — other MPD clients must not be
	// disrupted and the queue should remain for the next connection.
	select {
	case <-j.ctx.Done():
		j.state = stopped
		j.mu.Unlock()
		return nil
	default:
		if force {
			j.state = stopped
			j.queueLoaded = false
			j.queueTracks = nil
			j.mu.Unlock()
			return nil
		}
	}
	j.mu.Unlock()

	// Normal user-initiated stop — send commands to MPD.
	if err := j.provider.JukeboxStop(); err != nil {
		return err
	}
	if err := j.provider.JukeboxClear(); err != nil {
		return err
	}

	j.mu.Lock()
	j.queueLoaded = false
	j.queueTracks = nil
	j.queueLength = 0
	j.state = stopped
	j.mu.Unlock()

	j.InvokeOnStopped()
	return nil
}

func (j *JukeboxPlayer) PlayTrack(track *mediaprovider.Track, _ float64) error {
	// JukeboxSet replaces the entire MPD queue.
	if err := j.provider.JukeboxSet(track.ID); err != nil {
		return err
	}

	j.mu.Lock()
	j.startTrackTime = 0
	j.mu.Unlock()

	if err := j.startAndUpdateTime(); err != nil {
		return err
	}

	j.mu.Lock()
	j.queueLoaded = false
	j.queueTracks = nil
	j.curTrack = 0
	j.queueLength = 1
	j.curTrackDuration = track.Duration.Seconds()
	j.state = playing
	j.mu.Unlock()

	j.InvokeOnPlaying()
	j.InvokeOnTrackChange()
	return nil
}

func (j *JukeboxPlayer) SetNextTrack(track *mediaprovider.Track) error {
	if track == nil {
		return nil
	}

	j.mu.Lock()
	curTrack := j.curTrack
	queueLength := j.queueLength
	j.mu.Unlock()

	if curTrack < queueLength-1 {
		if err := j.provider.JukeboxRemove(curTrack + 1); err != nil {
			return err
		}
		j.mu.Lock()
		j.queueLength--
		j.mu.Unlock()
	}
	if err := j.provider.JukeboxAdd(track.ID); err != nil {
		return err
	}
	j.mu.Lock()
	j.queueLength++
	j.mu.Unlock()
	return nil
}

func (j *JukeboxPlayer) SeekSeconds(secs float64) error {
	j.mu.Lock()
	curTrack := j.curTrack
	j.seeking = true
	j.mu.Unlock()

	err := j.provider.JukeboxSeek(curTrack, int(secs))

	j.mu.Lock()
	j.seeking = false
	j.mu.Unlock()

	j.InvokeOnSeek()
	return err
}

func (j *JukeboxPlayer) IsSeeking() bool {
	j.mu.Lock()
	v := j.seeking
	j.mu.Unlock()
	return v
}

// GetStatus returns the last known playback status from the cache maintained by
// syncStatus.  It does NOT make a network call, keeping the hot poll path fast.
func (j *JukeboxPlayer) GetStatus() player.Status {
	j.mu.Lock()
	state := j.state
	pos := j.cachedTimePos
	j.mu.Unlock()

	var ps player.State
	switch state {
	case playing:
		ps = player.Playing
	case paused:
		ps = player.Paused
	default:
		ps = player.Stopped
	}

	return player.Status{State: ps, TimePos: pos}
}

func (j *JukeboxPlayer) Destroy() {
	if j.cancel != nil {
		j.cancel()
	}
}

// GetMediaInfo returns the cached audio info for the currently playing track.
func (j *JukeboxPlayer) GetMediaInfo() (MediaInfo, error) {
	j.mu.Lock()
	info := j.cachedAudioInfo
	j.mu.Unlock()
	return info, nil
}

// LoadQueue loads all tracks into the jukebox queue.
func (j *JukeboxPlayer) LoadQueue(tracks []*mediaprovider.Track) error {
	if err := j.provider.JukeboxClear(); err != nil {
		return err
	}
	for _, track := range tracks {
		if err := j.provider.JukeboxAdd(track.ID); err != nil {
			return err
		}
	}

	j.mu.Lock()
	j.queueLoaded = true
	j.queueTracks = tracks
	j.queueLength = len(tracks)
	j.curTrack = -1 // Not playing yet
	j.mu.Unlock()

	return nil
}

// PlayQueueIndex starts playback at the specified index in the loaded queue.
func (j *JukeboxPlayer) PlayQueueIndex(idx int) error {
	j.mu.Lock()
	queueLength := j.queueLength
	j.mu.Unlock()

	if idx < 0 || idx >= queueLength {
		return nil
	}
	if err := j.provider.JukeboxPlay(idx); err != nil {
		return err
	}

	j.mu.Lock()
	j.curTrack = idx
	if idx < len(j.queueTracks) {
		j.curTrackDuration = j.queueTracks[idx].Duration.Seconds()
	}
	j.state = playing
	j.mu.Unlock()

	j.InvokeOnPlaying()
	j.InvokeOnTrackChange()
	return nil
}

// IsQueueLoaded returns true if a queue has been loaded via LoadQueue.
func (j *JukeboxPlayer) IsQueueLoaded() bool {
	j.mu.Lock()
	v := j.queueLoaded
	j.mu.Unlock()
	return v
}

// ClearLoadedQueue clears the loaded queue state.
func (j *JukeboxPlayer) ClearLoadedQueue() {
	j.mu.Lock()
	j.queueLoaded = false
	j.queueTracks = nil
	j.mu.Unlock()
}

func (j *JukeboxPlayer) startAndUpdateTime() error {
	beforeStart := time.Now()
	if err := j.provider.JukeboxStart(); err != nil {
		return err
	}
	afterStart := time.Now()

	// assume track started playing at (i.e. has been playing for)
	// half the round-trip latency
	j.mu.Lock()
	j.startedAtUnixMilli = time.Now().Add(-afterStart.Sub(beforeStart)).UnixMilli()
	j.mu.Unlock()
	return nil
}
