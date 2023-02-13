package backend

import (
	"net/http"

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

func NewServerManager() *ServerManager {
	return &ServerManager{}
}

func (s *ServerManager) ConnectToServer(conf *ServerConfig, password string) error {
	cli := &subsonic.Client{
		Client:     &http.Client{},
		BaseUrl:    conf.Hostname,
		User:       conf.Username,
		ClientName: "supersonic",
	}
	if err := cli.Authenticate(password); err != nil {
		return err
	}
	s.Server = cli
	s.ServerID = conf.ID
	for _, cb := range s.onServerConnected {
		cb()
	}
	return nil
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
