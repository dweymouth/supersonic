package player

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/dweymouth/go-mpv"
)

// Error returned by many Player functions if called before the player has not been initialized.
var ErrUnitialized error = errors.New("mpv player uninitialized")

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
	PlaylistPos int64
}

// Argument to Seek function (SeekAbsolute, SeekRelative, SeekAbsolutePercent, SeekRelativePercent).
type SeekMode int

const (
	SeekAbsolute SeekMode = iota
	SeekRelative
	SeekAbsolutePercent
	SeekRelativePercent
)

// One of "no", "track", or "album"
type ReplayGainMode string

const (
	ReplayGainNone  ReplayGainMode = "no"
	ReplayGainTrack ReplayGainMode = "track"
	ReplayGainAlbum ReplayGainMode = "album"
)

// Replay Gain options (argument to SetReplayGainOptions).
type ReplayGainOptions struct {
	Mode            ReplayGainMode
	PreampGain      float64
	PreventClipping bool
	// Fallback gain intentionally omitted
}

// Player encapsulates the mpv instance and provides functions
// to control it and to check its status.
type Player struct {
	mpv            *mpv.Mpv
	initialized    bool
	vol            int
	replayGainOpts ReplayGainOptions
	haveRGainOpts  bool
	audioExclusive bool
	status         Status
	seeking        bool
	curPlaylistPos int64
	prePausedState State
	clientName     string

	bgCancel context.CancelFunc

	// callbacks
	onPaused      []func()
	onStopped     []func()
	onPlaying     []func()
	onSeek        []func()
	onTrackChange []func(int64)
}

// Returns a new player.
// Must call Init on the player before it is ready for playback.
func New() *Player {
	return NewWithClientName("")
}

// Same as New, but sets the application name that mpv
// reports to the system audio API.
func NewWithClientName(c string) *Player {
	return &Player{
		vol:        -1, // use 100 in Init
		clientName: c,
	}
}

// Initializes the Player and makes it ready for playback.
// Most Player functions will return ErrUnitialized if called before Init.
func (p *Player) Init(maxCacheMB int) error {
	if !p.initialized {
		m := mpv.Create()

		m.SetOptionString("idle", "yes")
		m.SetOptionString("video", "no")
		m.SetOptionString("audio-display", "no")
		m.SetOptionString("gapless-audio", "weak")
		m.SetOptionString("prefetch-playlist", "yes")
		m.SetOptionString("force-seekable", "yes")
		m.SetOptionString("terminal", "no")

		// limit in-memory cache size
		m.SetOptionString("demuxer-max-bytes", fmt.Sprintf("%dMiB", maxCacheMB))

		if p.vol < 0 {
			p.vol = 100
		}
		m.SetOption("volume", mpv.FORMAT_INT64, p.vol)

		p.SetAudioExclusive(p.audioExclusive)
		if p.haveRGainOpts {
			p.SetReplayGainOptions(p.replayGainOpts)
		}

		if p.clientName != "" {
			m.SetOptionString("audio-client-name", p.clientName)
		}

		if err := m.Initialize(); err != nil {
			return fmt.Errorf("error initializing mpv: %s", err.Error())
		}
		p.mpv = m
	}
	ctx, cancel := context.WithCancel(context.Background())
	go p.eventHandler(ctx)
	p.bgCancel = cancel
	p.initialized = true
	return nil
}

// Appends the given file to the play queue.
// Note that the Player API does not provide methods to read
// the play queue. Clients are expected to maintain their own play queue model.
func (p *Player) AppendFile(url string) error {
	if !p.initialized {
		return ErrUnitialized
	}
	return p.mpv.Command([]string{"loadfile", url, "append"})
}

// Plays the specified file, clearing the previous play queue, if any.
func (p *Player) PlayFile(url string) error {
	if !p.initialized {
		return ErrUnitialized
	}
	err := p.mpv.Command([]string{"loadfile", url, "replace"})
	if err == nil {
		p.setState(Playing)
	}
	return err
}

// Removes the item at the given index from the internal playqueue.
func (p *Player) RemoveTrackAt(idx int) error {
	if !p.initialized {
		return ErrUnitialized
	}
	return p.mpv.Command([]string{"playlist-remove", strconv.Itoa(idx)})
}

// Stops playback and clears the play queue.
func (p *Player) Stop() error {
	if !p.initialized {
		return ErrUnitialized
	}
	var err error
	if p.status.State == Stopped {
		err = p.mpv.Command([]string{"playlist-clear"})
	} else {
		if err = p.mpv.Command([]string{"stop"}); err == nil {
			// if player was paused, stop command actually doesn't clear pause state
			err = p.setPaused(false)
		}
	}
	if err == nil {
		p.setState(Stopped)
	}
	return err
}

// Clears the play queue, except for the currently playing file.
func (p *Player) ClearPlayQueue() error {
	if !p.initialized {
		return ErrUnitialized
	}
	return p.mpv.Command([]string{"playlist-clear"})
}

// Seeks within the currently playing track.
// See MPV seek command documentation for more details.
func (p *Player) Seek(target string, mode SeekMode) error {
	if !p.initialized {
		return ErrUnitialized
	}
	p.seeking = true
	err := p.mpv.Command([]string{"seek", target, mode.String()})
	return err
}

// Seeks to the beginning of the current track if:
//   - The current track is the first track in the play queue, or
//   - The current time is more than 3 seconds past the beginning of the track.
//
// Else seeks to the beginning of the previous track.
func (p *Player) SeekBackOrPrevious() error {
	if !p.initialized {
		return ErrUnitialized
	}

	if pos, err := p.getInt64Property("time-pos"); err == nil && pos > 3 {
		return p.Seek("0", SeekAbsolutePercent)
	}
	if pos, err := p.getInt64Property("playlist-pos"); err == nil && pos == 0 {
		return p.Seek("0", SeekAbsolutePercent)
	}
	return p.mpv.Command([]string{"playlist-prev"})
}

// Seeks to the next track in the play queue, if any.
func (p *Player) SeekNext() error {
	if !p.initialized {
		return ErrUnitialized
	}
	return p.mpv.Command([]string{"playlist-next"})
}

// Sets the volume of the player (0-100).
// Unlike most Player functions, SetVolume can be called before Init,
// to set the initial volume of the player on startup.
func (p *Player) SetVolume(vol int) error {
	if vol > 100 {
		vol = 100
	} else if vol < 0 {
		vol = 0
	}
	if p.initialized {
		err := p.mpv.SetProperty("volume", mpv.FORMAT_INT64, vol)
		if err == nil {
			p.vol = vol
		}
		return err
	}
	p.vol = vol
	return nil
}

// Sets the ReplayGain options of the player.
// Unlike most Player functions, SetReplayGainOptions can be called
// before Init, to set the initial replaygain options of the player on startup.
func (p *Player) SetReplayGainOptions(options ReplayGainOptions) error {
	p.replayGainOpts = options
	p.haveRGainOpts = true
	if p.initialized {
		if err := p.mpv.SetPropertyString("replaygain", string(options.Mode)); err != nil {
			return err
		}
		if err := p.mpv.SetProperty("replaygain-preamp", mpv.FORMAT_DOUBLE, options.PreampGain); err != nil {
			return err
		}
		clip := "no"
		if options.PreventClipping {
			clip = "yes"
		}
		if err := p.mpv.SetPropertyString("replaygain-clip", clip); err != nil {
			return err
		}
	}
	return nil
}

// Sets the audio exclusive option of the player.
// Unlike most Player functions, SetAudioExclusive can be called
// before Init, to set the initial option of the player on startup.
func (p *Player) SetAudioExclusive(tf bool) {
	p.audioExclusive = tf
	if p.initialized {
		val := "no"
		if tf {
			val = "yes"
		}
		p.mpv.SetOptionString("audio-exclusive", val)
	}
}

// Gets the current volume of the player.
func (p *Player) GetVolume() int {
	return p.vol
}

func (p *Player) setPaused(paused bool) error {
	return p.mpv.SetProperty("pause", mpv.FORMAT_FLAG, paused)
}

// Start playback from the first track in the play queue.
func (p *Player) PlayFromBeginning() error {
	return p.PlayTrackAt(0)
}

// Start playback from the specified track index in the play queue.
func (p *Player) PlayTrackAt(idx int) error {
	err := p.mpv.Command([]string{"playlist-play-index", strconv.Itoa(idx)})
	if p.GetStatus().State == Paused {
		err = p.setPaused(false)
	}
	if err == nil {
		p.setState(Playing)
	}
	return err
}

// Begins playback if there is anything in the play queue and player is stopped or paused.
// If player is playing, pauses playback.
func (p *Player) PlayPause() error {
	if !p.initialized {
		return ErrUnitialized
	}

	switch p.status.State {
	case Stopped:
		// check if we have anything to play
		if c, err := p.getInt64Property("playlist-count"); err == nil && c > 0 {
			err := p.mpv.Command([]string{"playlist-play-index", "0"})
			if err == nil {
				p.setState(Playing)
			}
			return err
		}
		return nil
	case Playing:
		err := p.setPaused(true)
		if err == nil {
			p.prePausedState = p.status.State
			p.setState(Paused)
		}
		return err
	case Paused:
		err := p.setPaused(false)
		if err == nil {
			p.setState(p.prePausedState)
		}
		return err
	default:
		return errors.New("Unknown player state")
	}
}

// Get the current status of the player.
func (p *Player) GetStatus() Status {
	if !p.initialized {
		return p.status
	}

	pos, _ := p.mpv.GetProperty("playback-time", mpv.FORMAT_DOUBLE)
	dur, _ := p.mpv.GetProperty("duration", mpv.FORMAT_DOUBLE)
	if pos != nil {
		p.status.TimePos = pos.(float64)
	}
	if dur != nil {
		p.status.Duration = dur.(float64)
	}
	if playpos, err := p.getInt64Property("playlist-pos"); err == nil {
		p.status.PlaylistPos = playpos
	}
	return p.status
}

func (p *Player) getInt64Property(propName string) (int64, error) {
	playpos, err := p.mpv.GetProperty(propName, mpv.FORMAT_INT64)
	if err != nil {
		return -1, err
	}
	if playpos != nil {
		return playpos.(int64), nil
	}
	return -1, errors.New("mpv did not report playlist pos")
}

// Returns true if a seek is currently in progress.
func (p *Player) IsSeeking() bool {
	return p.seeking && p.status.State == Playing
}

// Registers a callback which is invoked when the player transitions to the Paused state.
func (p *Player) OnPaused(cb func()) {
	p.onPaused = append(p.onPaused, cb)
}

// Registers a callback which is invoked when the player transitions to the Stopped state.
func (p *Player) OnStopped(cb func()) {
	p.onStopped = append(p.onStopped, cb)
}

// Registers a callback which is invoked when the player transitions to the Playing state.
func (p *Player) OnPlaying(cb func()) {
	p.onPlaying = append(p.onPlaying, cb)
}

// Registers a callback which is invoked whenever a seek event occurs.
func (p *Player) OnSeek(cb func()) {
	p.onSeek = append(p.onSeek, cb)
}

// Registers a callback which is invoked when the currently playing track changes,
// or when playback begins at any time from the Stopped state.
// Callback is invoked with the index of the currently playing track (zero-based).
func (p *Player) OnTrackChange(cb func(int64)) {
	p.onTrackChange = append(p.onTrackChange, cb)
}

// Destroy the player.
func (p *Player) Destroy() {
	if p.bgCancel != nil {
		p.bgCancel()
	}
	if p.initialized {
		p.mpv.Command([]string{"stop"})
		p.mpv.TerminateDestroy()
		p.initialized = false
	}
}

// sets the state and invokes callbacks, if triggered
func (p *Player) setState(s State) {
	switch {
	case s == Playing && p.status.State != Playing:
		defer func() {
			for _, cb := range p.onPlaying {
				cb()
			}
		}()
	case s == Paused && p.status.State != Paused:
		defer func() {
			for _, cb := range p.onPaused {
				cb()
			}
		}()
	case s == Stopped && p.status.State != Stopped:
		defer func() {
			for _, cb := range p.onStopped {
				cb()
			}
		}()
	}
	p.status.State = s
}

func (p *Player) eventHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			e := p.mpv.WaitEvent(1 /*timeout seconds*/)
			if e.Event_Id != mpv.EVENT_NONE {
				//log.Printf("mpv event: %+v\n", e)
			}
			switch e.Event_Id {
			case mpv.EVENT_PLAYBACK_RESTART:
				if p.seeking {
					p.seeking = false
				}
			case mpv.EVENT_SEEK:
				for _, cb := range p.onSeek {
					cb()
				}
			case mpv.EVENT_FILE_LOADED:
				if p.status.State == Paused {
					// seek while paused switches to a new file
					// mpv does not fire seek event in this case
					for _, cb := range p.onSeek {
						cb()
					}
				}
				if pos, err := p.getInt64Property("playlist-pos"); err == nil {
					p.curPlaylistPos = pos
					for _, cb := range p.onTrackChange {
						cb(pos)
					}
				}
			case mpv.EVENT_IDLE:
				p.status.Duration = 0
				p.status.TimePos = 0
				p.setState(Stopped)
			}
		}
	}
}

func (s SeekMode) String() string {
	switch s {
	case SeekAbsolute:
		return "absolute"
	case SeekRelative:
		return "relative"
	case SeekAbsolutePercent:
		return "absolute-percent"
	case SeekRelativePercent:
		return "relative-percent"
	}
	return "UNKNOWN_SEEK_MODE"
}
