package player

type URLPlayer interface {
	BasePlayer
	AppendFile(url string) error
	PlayFile(url string) error
}

type TrackIDPlayer interface {
	BasePlayer
	AppendTrack(trackID string) error
	PlayTrack(trackID string) error
}

type BasePlayer interface {
	// Transport
	PlayTrackAt(idx int) error
	Continue() error
	Pause() error
	PlayPause() error
	Stop() error
	SeekPrevious() error
	SeekNext() error
	SeekSeconds(secs float64) error
	IsSeeking() bool

	SetVolume(int) error
	GetVolume() int

	GetStatus() Status

	ClearPlayQueue() error
	RemoveTrackAt(idx int) error

	SetLoopMode(LoopMode) error
	GetLoopMode() LoopMode

	// Event API
	OnPaused(func())
	OnStopped(func())
	OnPlaying(func())
	OnSeek(func())
	OnTrackChange(func(int))
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
	State       State
	TimePos     float64
	Duration    float64
	PlaylistPos int
}

// The playback loop mode (LoopNone, LoopAll, LoopOne).
type LoopMode int

const (
	LoopNone LoopMode = iota
	LoopAll
	LoopOne
)

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

func (l LoopMode) String() string {
	switch l {
	case LoopNone:
		return "no"
	case LoopAll:
		return "all"
	case LoopOne:
		return "one"
	}
	return "UNKNOWN_LOOP_MODE"
}
