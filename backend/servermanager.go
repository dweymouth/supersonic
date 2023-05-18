package backend

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	subsonicMP "github.com/dweymouth/supersonic/backend/mediaprovider/subsonic"
	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
)

type ServerManager struct {
	LoggedInUser string
	ServerID     uuid.UUID
	Server       mediaprovider.MediaProvider

	prefetchCoverCB   func(string)
	appName           string
	config            *Config
	onServerConnected []func()
	onLogout          []func()
}

var ErrUnreachable = errors.New("server is unreachable")

func NewServerManager(appName string, config *Config) *ServerManager {
	return &ServerManager{appName: appName, config: config}
}

func (s *ServerManager) SetPrefetchAlbumCoverCallback(cb func(string)) {
	s.prefetchCoverCB = cb
	if s.Server != nil {
		s.Server.SetPrefetchCoverCallback(cb)
	}
}

func (s *ServerManager) ConnectToServer(conf *ServerConfig, password string) error {
	cli, err := s.testConnectionAndCreateClient(conf.ServerConnection, password)
	if err != nil {
		return err
	}
	s.Server = subsonicMP.SubsonicMediaProvider(cli)
	s.Server.SetPrefetchCoverCallback(s.prefetchCoverCB)
	s.LoggedInUser = conf.Username
	s.ServerID = conf.ID
	s.SetDefaultServer(s.ServerID)
	for _, cb := range s.onServerConnected {
		cb()
	}
	return nil
}

func (s *ServerManager) TestConnectionAndAuth(
	connection ServerConnection, password string, timeout time.Duration,
) error {
	err := ErrUnreachable
	done := make(chan bool)
	go func() {
		_, err = s.testConnectionAndCreateClient(connection, password)
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

func (s *ServerManager) GetDefaultServer() *ServerConfig {
	for _, s := range s.config.Servers {
		if s.Default {
			return s
		}
	}
	if len(s.config.Servers) > 0 {
		return s.config.Servers[0]
	}
	return nil
}

func (s *ServerManager) SetDefaultServer(serverID uuid.UUID) {
	var found bool
	for _, s := range s.config.Servers {
		f := s.ID == serverID
		if f {
			found = true
		}
		s.Default = f
	}
	if !found && len(s.config.Servers) > 0 {
		s.config.Servers[0].Default = true
	}
}

func (s *ServerManager) AddServer(nickname string, connection ServerConnection) *ServerConfig {
	sc := &ServerConfig{
		ID:               uuid.New(),
		Nickname:         nickname,
		ServerConnection: connection,
	}
	s.config.Servers = append(s.config.Servers, sc)
	return sc
}

func (s *ServerManager) DeleteServer(serverID uuid.UUID) {
	s.deleteServerPassword(serverID)
	newServers := make([]*ServerConfig, 0, len(s.config.Servers)-1)
	for _, s := range s.config.Servers {
		if s.ID != serverID {
			newServers = append(newServers, s)
		}
	}
	s.config.Servers = newServers
}

func (s *ServerManager) Logout(deletePassword bool) {
	if s.Server != nil {
		if deletePassword {
			s.deleteServerPassword(s.ServerID)
		}
		for _, cb := range s.onLogout {
			cb()
		}
		s.Server = nil
		s.LoggedInUser = ""
		s.ServerID = uuid.UUID{}
	}
}

func (s *ServerManager) deleteServerPassword(serverID uuid.UUID) {
	keyring.Delete(s.appName, s.ServerID.String())
}

// Sets a callback that is invoked when a server is connected to.
func (s *ServerManager) OnServerConnected(cb func()) {
	s.onServerConnected = append(s.onServerConnected, cb)
}

// Sets a callback that is invoked when the user logs out of a server.
func (s *ServerManager) OnLogout(cb func()) {
	s.onLogout = append(s.onLogout, cb)
}

func (s *ServerManager) GetServerPassword(serverID uuid.UUID) (string, error) {
	return keyring.Get(s.appName, serverID.String())
}

func (s *ServerManager) SetServerPassword(server *ServerConfig, password string) error {
	return keyring.Set(s.appName, server.ID.String(), password)
}

func (s *ServerManager) testConnectionAndCreateClient(connection ServerConnection, password string) (*subsonic.Client, error) {
	cli, err := s.connect(connection, password)
	if err != nil {
		return nil, err
	}
	if err := cli.Authenticate(password); err != nil {
		return nil, err
	}
	return cli, nil
}

func (s *ServerManager) connect(connection ServerConnection, password string) (*subsonic.Client, error) {
	cli := &subsonic.Client{
		Client:       &http.Client{Timeout: 10 * time.Second},
		BaseUrl:      connection.Hostname,
		User:         connection.Username,
		PasswordAuth: connection.LegacyAuth,
		ClientName:   "supersonic",
	}
	altCli := &subsonic.Client{
		Client:       &http.Client{Timeout: 10 * time.Second},
		BaseUrl:      connection.AltHostname,
		User:         connection.Username,
		PasswordAuth: connection.LegacyAuth,
		ClientName:   "supersonic",
	}
	pingChan := make(chan bool, 2) // false for primary hostname, true for alternate
	pingFunc := func(delay time.Duration, cli *subsonic.Client, val bool) {
		<-time.After(delay)
		if cli.Ping() {
			pingChan <- val
		}
	}
	go pingFunc(0, cli, false)
	if connection.AltHostname != "" {
		go pingFunc(333*time.Millisecond, altCli, true) // give primary hostname ping a head start
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		return nil, ErrUnreachable
	case altPing := <-pingChan:
		if altPing {
			return altCli, nil
		}
		return cli, nil
	}
}
