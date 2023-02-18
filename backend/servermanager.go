package backend

import (
	"errors"
	"net/http"
	"time"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
)

type ServerManager struct {
	ServerID uuid.UUID
	Server   *subsonic.Client

	onServerConnected []func()
	onLogout          []func()
}

var ErrUnreachable = errors.New("server is unreachable")

func NewServerManager() *ServerManager {
	return &ServerManager{}
}

func (s *ServerManager) ConnectToServer(conf *ServerConfig, password string) error {
	cli, err := s.testConnectionAndCreateClient(conf.Hostname, conf.Username, password)
	if err != nil {
		return err
	}
	s.Server = cli
	s.ServerID = conf.ID
	for _, cb := range s.onServerConnected {
		cb()
	}
	return nil
}

func (s *ServerManager) TestConnectionAndAuth(hostname, username, password string, timeout time.Duration) error {
	err := ErrUnreachable
	done := make(chan bool)
	go func() {
		_, err = s.testConnectionAndCreateClient(hostname, username, password)
		close(done)
	}()
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-t.C:
		return err
	case <-done:
		return err
	}
}

func (s *ServerManager) testConnectionAndCreateClient(hostname, username, password string) (*subsonic.Client, error) {
	cli := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    hostname,
		User:       username,
		ClientName: "supersonic",
	}
	if !cli.Ping() {
		return nil, ErrUnreachable
	}
	if err := cli.Authenticate(password); err != nil {
		return nil, err
	}
	return cli, nil
}

func (s *ServerManager) Logout() {
	if s.Server != nil {
		keyring.Delete(AppName, s.ServerID.String())
		for _, cb := range s.onLogout {
			cb()
		}
		s.Server = nil
		s.ServerID = uuid.UUID{}
	}
}

func (s *ServerManager) OnServerConnected(cb func()) {
	s.onServerConnected = append(s.onServerConnected, cb)
}

func (s *ServerManager) OnLogout(cb func()) {
	s.onLogout = append(s.onLogout, cb)
}

func (s *ServerManager) GetServerPassword(server *ServerConfig) (string, error) {
	return keyring.Get(AppName, server.ID.String())
}

func (s *ServerManager) SetServerPassword(server *ServerConfig, password string) error {
	return keyring.Set(AppName, server.ID.String(), password)
}
