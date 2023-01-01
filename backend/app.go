package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"supersonic/player"

	"github.com/20after4/configdir"
	"github.com/zalando/go-keyring"
)

const (
	AppName    = "supersonic"
	configFile = "config.toml"
)

var (
	ErrNoServers = errors.New("no servers set up")
)

type App struct {
	Config          *Config
	ServerManager   *ServerManager
	ImageManager    *ImageManager
	LibraryManager  *LibraryManager
	PlaybackManager *PlaybackManager
	Player          *player.Player

	bgrndCtx context.Context
	cancel   context.CancelFunc
}

func StartupApp() (*App, error) {
	a := &App{}
	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())

	log.Printf("Starting %s...", AppName)
	log.Printf("Using config dir: %s", configdir.LocalConfig(AppName))
	log.Printf("Using cache dir: %s", configdir.LocalCache(AppName))

	a.readConfig()

	if err := a.initMPV(); err != nil {
		return nil, err
	}

	a.ServerManager = NewServerManager()
	a.PlaybackManager = NewPlaybackManager(a.bgrndCtx, a.ServerManager, a.Player)
	a.LibraryManager = NewLibraryManager(a.ServerManager)
	a.ImageManager = NewImageManager(a.ServerManager, configdir.LocalCache(AppName))

	return a, nil
}

func (a *App) readConfig() {
	configdir.MakePath(configdir.LocalConfig(AppName))
	cfg, err := ReadConfigFile(configPath())
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = DefaultConfig()
	}
	a.Config = cfg
}

func (a *App) initMPV() error {
	p := player.NewWithClientName(AppName)
	if err := p.Init(); err != nil {
		return fmt.Errorf("failed to initialize mpv player: %s", err.Error())
	}
	a.Player = p
	return nil
}

func (a *App) LoginToDefaultServer() error {
	serverCfg := a.Config.GetDefaultServer()
	if serverCfg == nil {
		return ErrNoServers
	}
	pass, err := keyring.Get(AppName, serverCfg.ID.String())
	if err != nil {
		return fmt.Errorf("error reading keyring credentials: %v", err)
	}
	return a.ServerManager.ConnectToServer(serverCfg, pass)
}

func (a *App) Shutdown() {
	a.Player.Stop()
	a.cancel()
	a.Player.Destroy()
}

func configPath() string {
	return path.Join(configdir.LocalConfig(AppName), configFile)
}
