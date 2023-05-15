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

	appName           string
	onServerConnected []func()
	onLogout          []func()
}

var ErrUnreachable = errors.New("server is unreachable")

func NewServerManager(appName string) *ServerManager {
	return &ServerManager{appName: appName}
}

func (s *ServerManager) ConnectToServer(conf *ServerConfig, password string) error {
	cli, err := s.testConnectionAndCreateClient(conf.ServerConnection, password)
	if err != nil {
		return err
	}
	s.Server = subsonicMP.SubsonicMediaProvider(cli)
	s.LoggedInUser = conf.Username
	s.ServerID = conf.ID
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

func (s *ServerManager) Logout() {
	if s.Server != nil {
		keyring.Delete(s.appName, s.ServerID.String())
		for _, cb := range s.onLogout {
			cb()
		}
		s.Server = nil
		s.LoggedInUser = ""
		s.ServerID = uuid.UUID{}
	}
}

// Sets a callback that is invoked when a server is connected to.
func (s *ServerManager) OnServerConnected(cb func()) {
	s.onServerConnected = append(s.onServerConnected, cb)
}

// Sets a callback that is invoked when the user logs out of a server.
func (s *ServerManager) OnLogout(cb func()) {
	s.onLogout = append(s.onLogout, cb)
}

func (s *ServerManager) GetServerPassword(server *ServerConfig) (string, error) {
	return keyring.Get(s.appName, server.ID.String())
}

func (s *ServerManager) SetServerPassword(server *ServerConfig, password string) error {
	return keyring.Set(s.appName, server.ID.String(), password)
}
