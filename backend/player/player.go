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
