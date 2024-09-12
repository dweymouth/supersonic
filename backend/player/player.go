package player

import "github.com/dweymouth/supersonic/backend/mediaprovider"

type URLPlayer interface {
	BasePlayer
	PlayFile(url string) error
	SetNextFile(url string) error
}

type TrackPlayer interface {
	BasePlayer
	PlayTrack(track *mediaprovider.Track) error
	SetNextTrack(track *mediaprovider.Track) error
}

type BasePlayer interface {
	Continue() error
	Pause() error
	Stop() error

	SeekSeconds(secs float64) error
	IsSeeking() bool

	SetVolume(int) error
	GetVolume() int

	GetStatus() Status

	// Event API
	OnPaused(func())
	OnStopped(func())
	OnPlaying(func())
	OnSeek(func())
	OnTrackChange(func())
}

type ReplayGainPlayer interface {
	SetReplayGainOptions(ReplayGainOptions) error
}

// The playback state (Stopped, Paused, or Playing).
type State int

const (
	Stopped State = iota
	Paused
	Playing
)

// The current status of the player.
// Includes playback state, current time, total track time, and playlist position.
type Status struct {
	State    State
	TimePos  float64
	Duration float64
}

type ReplayGainMode int

const (
	ReplayGainNone ReplayGainMode = iota
	ReplayGainTrack
	ReplayGainAlbum
)

// Replay Gain options (argument to SetReplayGainOptions).
type ReplayGainOptions struct {
	Mode            ReplayGainMode
	PreampGain      float64
	PreventClipping bool
	// Fallback gain intentionally omitted
}

func (r ReplayGainMode) String() string {
	switch r {
	case ReplayGainTrack:
		return "track"
	case ReplayGainAlbum:
		return "album"
	default:
		return "no"
	}
}

type BasePlayerCallbackImpl struct {
	onPaused      []func()
	onStopped     []func()
	onPlaying     []func()
	onSeek        []func()
	onTrackChange []func()
}

// Registers a callback which is invoked when the player transitions to the Paused state.
func (p *BasePlayerCallbackImpl) OnPaused(cb func()) {
	p.onPaused = append(p.onPaused, cb)
}

// Registers a callback which is invoked when the player transitions to the Stopped state.
func (p *BasePlayerCallbackImpl) OnStopped(cb func()) {
	p.onStopped = append(p.onStopped, cb)
}

// Registers a callback which is invoked when the player transitions to the Playing state.
func (p *BasePlayerCallbackImpl) OnPlaying(cb func()) {
	p.onPlaying = append(p.onPlaying, cb)
}

// Registers a callback which is invoked whenever a seek event occurs.
func (p *BasePlayerCallbackImpl) OnSeek(cb func()) {
	p.onSeek = append(p.onSeek, cb)
}

// Registers a callback which is invoked when the currently playing track changes,
// or when playback begins at any time from the Stopped state.
// Callback is invoked with the index of the currently playing track (zero-based).
func (p *BasePlayerCallbackImpl) OnTrackChange(cb func()) {
	p.onTrackChange = append(p.onTrackChange, cb)
}

func (p *BasePlayerCallbackImpl) InvokeOnPaused() {
	for _, cb := range p.onPaused {
		cb()
	}
}

func (p *BasePlayerCallbackImpl) InvokeOnPlaying() {
	for _, cb := range p.onPlaying {
		cb()
	}
}

func (p *BasePlayerCallbackImpl) InvokeOnStopped() {
	for _, cb := range p.onStopped {
		cb()
	}
}

func (p *BasePlayerCallbackImpl) InvokeOnSeek() {
	for _, cb := range p.onSeek {
		cb()
	}
}

func (p *BasePlayerCallbackImpl) InvokeOnTrackChange() {
	for _, cb := range p.onTrackChange {
		cb()
	}
}
