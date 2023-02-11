package backend

import (
	"net/http"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/google/uuid"
)

type ServerManager struct {
	ServerID uuid.UUID
	Server   *subsonic.Client

	onServerConnected []func()
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

func (s *ServerManager) OnServerConnected(cb func()) {
	s.onServerConnected = append(s.onServerConnected, cb)
}
