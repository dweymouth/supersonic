package backend

import (
	"encoding/base32"
	"errors"
	"fmt"
	"strconv"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
	"github.com/godbus/dbus/v5"
	"github.com/quarckster/go-mpris-server/pkg/events"
	"github.com/quarckster/go-mpris-server/pkg/server"
	"github.com/quarckster/go-mpris-server/pkg/types"
)

const (
	dbusTrackIDPrefix = "/Supersonic/Track/"
	noTrackObjectPath = "/org/mpris/MediaPlayer2/TrackList/NoTrack"
)

var (
	_ types.OrgMprisMediaPlayer2Adapter                 = (*MPRISHandler)(nil)
	_ types.OrgMprisMediaPlayer2PlayerAdapter           = (*MPRISHandler)(nil)
	_ types.OrgMprisMediaPlayer2PlayerAdapterLoopStatus = (*MPRISHandler)(nil)
)

var (
	errNotSupported = errors.New("not supported")
)

type MPRISHandler struct {
	// Function called if the player is requested to quit through MPRIS.
	// Should *asynchronously* start shutdown and return immediately true if a shutdown will happen.
	OnQuit func() error

	// Function called if the player is requested to bring its UI to the front.
	OnRaise func() error

	// Function to look up the artwork URL for a given track ID
	ArtURLLookup func(trackID string) (string, error)

	connErr      error
	playerName   string
	curTrackPath string // empty for no track
	p            *player.Player
	pm           *PlaybackManager
	s            *server.Server
	evt          *events.EventHandler
}

func NewMPRISHandler(playerName string, p *player.Player, pm *PlaybackManager) *MPRISHandler {
	m := &MPRISHandler{playerName: playerName, p: p, pm: pm, connErr: errors.New("not started")}
	m.s = server.NewServer(playerName, m, m)
	m.evt = events.NewEventHandler(m.s)

	m.p.OnSeek(func() {
		if m.connErr == nil {
			pos := secondsToMicroseconds(m.p.GetStatus().TimePos)
			m.evt.Player.OnSeek(pos)
		}
	})
	m.pm.OnSongChange(func(tr, _ *mediaprovider.Track) {
		if m.connErr == nil {
			m.evt.Player.OnTitle()
		}
		if tr == nil {
			m.curTrackPath = ""
		} else {
			m.curTrackPath = dbusTrackIDPrefix + encodeTrackId(tr.ID)
		}
	})
	m.pm.OnVolumeChange(func(vol int) {
		if m.connErr == nil {
			m.evt.Player.OnVolume()
		}
	})
	emitPlayStatus := func() {
		if m.connErr == nil {
			m.evt.Player.OnPlayPause()
		}
	}
	m.p.OnStopped(emitPlayStatus)
	m.p.OnPlaying(emitPlayStatus)
	m.p.OnPaused(emitPlayStatus)

	return m
}

// Starts listening for MPRIS events.
func (m *MPRISHandler) Start() {
	m.connErr = nil
	go func() {
		// exits early with err if unable to establish D-Bus connection
		m.connErr = m.s.Listen()
	}()
}

// Stops listening for MPRIS events and releases any D-Bus resources.
func (m *MPRISHandler) Shutdown() {
	if m.connErr == nil {
		m.s.Stop()
		m.connErr = errors.New("stopped")
	}
}

// OrgMprisMediaPlayer2Adapter implementation

func (m *MPRISHandler) Identity() (string, error) {
	return m.playerName, nil
}

func (m *MPRISHandler) CanQuit() (bool, error) {
	return m.OnQuit != nil, nil
}

func (m *MPRISHandler) Quit() error {
	if m.OnQuit != nil {
		return m.OnQuit()
	}
	return errors.New("no quit handler added")
}

func (m *MPRISHandler) CanRaise() (bool, error) {
	return m.OnRaise != nil, nil
}

func (m *MPRISHandler) Raise() error {
	if m.OnRaise != nil {
		return m.OnRaise()
	}
	return errors.New("no raise handler added")
}

func (m *MPRISHandler) HasTrackList() (bool, error) {
	return false, nil
}

func (m *MPRISHandler) SupportedUriSchemes() ([]string, error) {
	return nil, nil
}

func (m *MPRISHandler) SupportedMimeTypes() ([]string, error) {
	return nil, nil
}

// OrgMprisMediaPlayer2PlayerAdapter implementation

func (m *MPRISHandler) Next() error {
	return m.p.SeekNext()
}

func (m *MPRISHandler) Previous() error {
	return m.p.SeekBackOrPrevious()
}

func (m *MPRISHandler) Pause() error {
	if m.p.GetStatus().State == player.Playing {
		return m.p.PlayPause()
	}
	return nil
}

func (m *MPRISHandler) PlayPause() error {
	return m.p.PlayPause()
}

func (m *MPRISHandler) Stop() error {
	return m.p.Stop()
}

func (m *MPRISHandler) Play() error {
	switch m.p.GetStatus().State {
	case player.Paused:
		return m.p.PlayPause()
	case player.Stopped:
		return m.p.PlayFromBeginning()
	}
	return nil
}

func (m *MPRISHandler) Seek(offset types.Microseconds) error {
	return m.p.Seek(fmt.Sprintf("%0.2f", microsecondsToSeconds(offset)), player.SeekRelative)
}

func (m *MPRISHandler) SetPosition(trackId string, position types.Microseconds) error {
	if m.curTrackPath == trackId {
		return m.p.Seek(fmt.Sprintf("%0.2f", microsecondsToSeconds(position)), player.SeekAbsolute)
	}
	return nil
}

func (m *MPRISHandler) OpenUri(uri string) error {
	return errNotSupported
}

func (m *MPRISHandler) PlaybackStatus() (types.PlaybackStatus, error) {
	switch m.p.GetStatus().State {
	case player.Playing:
		return types.PlaybackStatusPlaying, nil
	case player.Paused:
		return types.PlaybackStatusPaused, nil
	case player.Stopped:
		return types.PlaybackStatusStopped, nil
	}
	return "", errors.New("unknown playback status")
}

func (m *MPRISHandler) LoopStatus() (types.LoopStatus, error) {
	switch m.pm.LoopMode() {
	case LoopModeAll:
		return types.LoopStatusPlaylist, nil
	case LoopModeOne:
		return types.LoopStatusTrack, nil
	case LoopModeNone:
		return types.LoopStatusNone, nil
	}
	return "", errors.New("unknown loop status")
}

func (m *MPRISHandler) SetLoopStatus(status types.LoopStatus) error {
	switch status {
	case types.LoopStatusPlaylist:
		return m.pm.SetLoopMode(LoopModeAll)
	case types.LoopStatusTrack:
		return m.pm.SetLoopMode(LoopModeOne)
	case types.LoopStatusNone:
		return m.pm.SetLoopMode(LoopModeNone)
	}
	return errors.New("unknown loop status")
}

func (m *MPRISHandler) Rate() (float64, error) {
	return 1, nil
}

func (m *MPRISHandler) SetRate(float64) error {
	return errNotSupported
}

func (m *MPRISHandler) Metadata() (types.Metadata, error) {
	trackObjPath := noTrackObjectPath
	if m.curTrackPath != "" {
		trackObjPath = m.curTrackPath
	}
	status := m.p.GetStatus()
	var tr mediaprovider.Track
	if np := m.pm.NowPlaying(); np != nil && status.State != player.Stopped {
		tr = *np
	}
	var artURL string
	if tr.ID != "" && m.ArtURLLookup != nil {
		if u, err := m.ArtURLLookup(tr.CoverArtID); err == nil {
			artURL = u
		}
	}
	return types.Metadata{
		TrackId:        dbus.ObjectPath(trackObjPath),
		Length:         secondsToMicroseconds(status.Duration),
		Title:          tr.Name,
		Album:          tr.Album,
		Artist:         tr.ArtistNames,
		DiscNumber:     tr.DiscNumber,
		TrackNumber:    tr.TrackNumber,
		Genre:          []string{tr.Genre},
		UserRating:     float64(tr.Rating) / 5,
		ContentCreated: strconv.Itoa(tr.Year),
		UseCount:       tr.PlayCount,
		ArtUrl:         artURL,
	}, nil
}

func (m *MPRISHandler) Volume() (float64, error) {
	return float64(m.p.GetVolume()) / 100, nil
}

func (m *MPRISHandler) SetVolume(v float64) error {
	return m.pm.SetVolume(int(v * 100))
}

func (m *MPRISHandler) Position() (int64, error) {
	return int64(secondsToMicroseconds(m.p.GetStatus().TimePos)), nil
}

func (m *MPRISHandler) MinimumRate() (float64, error) {
	return 1, nil
}

func (m *MPRISHandler) MaximumRate() (float64, error) {
	return 1, nil
}

func (m *MPRISHandler) CanGoNext() (bool, error) {
	return true, nil
}

func (m *MPRISHandler) CanGoPrevious() (bool, error) {
	return true, nil
}

func (m *MPRISHandler) CanPlay() (bool, error) {
	return true, nil
}

func (m *MPRISHandler) CanPause() (bool, error) {
	return true, nil
}

func (m *MPRISHandler) CanSeek() (bool, error) {
	return true, nil
}

func (m *MPRISHandler) CanControl() (bool, error) {
	return true, nil
}

func microsecondsToSeconds(m types.Microseconds) float64 {
	return float64(m) / 1_000_000
}

func secondsToMicroseconds(s float64) types.Microseconds {
	return types.Microseconds(s * 1_000_000)
}

func encodeTrackId(id string) string {
	data := []byte(id)
	return base32.StdEncoding.WithPadding('0').EncodeToString(data)
}

