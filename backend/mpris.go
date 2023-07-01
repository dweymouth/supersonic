package backend

import (
	"errors"

	"github.com/dweymouth/supersonic/player"
	"github.com/quarckster/go-mpris-server/pkg/events"
	"github.com/quarckster/go-mpris-server/pkg/server"
	"github.com/quarckster/go-mpris-server/pkg/types"
)

var (
	_ types.OrgMprisMediaPlayer2Adapter       = (*MPRISHandler)(nil)
	_ types.OrgMprisMediaPlayer2PlayerAdapter = (*MPRISHandler)(nil)
)
var (
	notImplemented = errors.New("not implemented")
)

type MPRISHandler struct {
	p   *player.Player
	pm  *PlaybackManager
	s   *server.Server
	evt *events.EventHandler
}

func NewMPRISHandler(p *player.Player, pm *PlaybackManager) *MPRISHandler {
	m := &MPRISHandler{p: p, pm: pm}
	m.s = server.NewServer("Supersonic", m, m)
	m.evt = events.NewEventHandler(m.s)
	return m
}

func (m *MPRISHandler) Start() {
	go m.s.Listen()
}

func (m *MPRISHandler) Shutdown() {
	m.s.Stop()
}

// OrgMprisMediaPlayer2Adapter implementation

func (m *MPRISHandler) Identity() (string, error) {
	return "supersonic", nil
}

func (m *MPRISHandler) CanQuit() (bool, error) {
	return false, nil
}

func (m *MPRISHandler) Quit() error {
	return errors.New("not implemented")
}

func (m *MPRISHandler) CanRaise() (bool, error) {
	return false, nil
}

func (m *MPRISHandler) Raise() error {
	return errors.New("not implemented")
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
	return notImplemented
}

func (m *MPRISHandler) PlayPause() error {
	return m.p.PlayPause()
}

func (m *MPRISHandler) Stop() error {
	return notImplemented
}

func (m *MPRISHandler) Play() error {
	return notImplemented
}

func (m *MPRISHandler) Seek(offset types.Microseconds) error {
	return notImplemented
}

func (m *MPRISHandler) SetPosition(trackId string, position types.Microseconds) error {
	return notImplemented
}

func (m *MPRISHandler) OpenUri(uri string) error {
	return notImplemented
}

func (m *MPRISHandler) PlaybackStatus() (types.PlaybackStatus, error) {
	return "", notImplemented
}

func (m *MPRISHandler) Rate() (float64, error) {
	return 0, notImplemented
}

func (m *MPRISHandler) SetRate(float64) error {
	return notImplemented
}

func (m *MPRISHandler) Metadata() (types.Metadata, error) {
	return types.Metadata{}, notImplemented
}

func (m *MPRISHandler) Volume() (float64, error) {
	return 0, notImplemented
}

func (m *MPRISHandler) SetVolume(float64) error {
	return notImplemented
}

func (m *MPRISHandler) Position() (int64, error) {
	return 0, notImplemented
}

func (m *MPRISHandler) MinimumRate() (float64, error) {
	return 0, notImplemented
}

func (m *MPRISHandler) MaximumRate() (float64, error) {
	return 0, notImplemented
}

func (m *MPRISHandler) CanGoNext() (bool, error) {
	return false, notImplemented
}

func (m *MPRISHandler) CanGoPrevious() (bool, error) {
	return false, notImplemented
}

func (m *MPRISHandler) CanPlay() (bool, error) {
	return false, notImplemented
}

func (m *MPRISHandler) CanPause() (bool, error) {
	return false, notImplemented
}

func (m *MPRISHandler) CanSeek() (bool, error) {
	return false, notImplemented
}

func (m *MPRISHandler) CanControl() (bool, error) {
	return false, notImplemented
}
