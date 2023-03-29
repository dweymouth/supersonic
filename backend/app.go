package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"supersonic/backend/util"
	"supersonic/player"
	"supersonic/sharedutil"

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
	a.Config.LocalPlayback.Volume = clamp(a.Config.LocalPlayback.Volume, 0, 100)
	a.Player.SetVolume(a.Config.LocalPlayback.Volume)

	rgainOpts := []string{ReplayGainNone, ReplayGainAlbum, ReplayGainTrack}
	if !sharedutil.StringSliceContains(rgainOpts, a.Config.ReplayGain.Mode) {
		a.Config.ReplayGain.Mode = ReplayGainNone
	}
	a.Player.SetReplayGainOptions(player.ReplayGainOptions{
		Mode:            player.ReplayGainMode(a.Config.ReplayGain.Mode),
		PreventClipping: a.Config.ReplayGain.PreventClipping,
		PreampGain:      a.Config.ReplayGain.PreampGainDB,
	})

	a.ServerManager = NewServerManager()
	a.PlaybackManager = NewPlaybackManager(a.bgrndCtx, a.ServerManager, a.Player, &a.Config.Scrobbling)
	a.LibraryManager = NewLibraryManager(a.ServerManager)
	a.ImageManager = NewImageManager(a.bgrndCtx, a.ServerManager, configdir.LocalCache(AppName))
	a.LibraryManager.PreCacheCoverFn = func(coverID string) {
		_, _ = a.ImageManager.GetAlbumThumbnail(coverID)
	}

	return a, nil
}

func (a *App) readConfig() {
	configdir.MakePath(configdir.LocalConfig(AppName))
	cfgPath := configPath()
	cfg, err := ReadConfigFile(cfgPath)
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = DefaultConfig()
		if _, err := os.Stat(cfgPath); err == nil {
			backupCfgName := fmt.Sprintf("%s.bak", configFile)
			log.Printf("Config file may be malformed: copying to %s", backupCfgName)
			_ = util.CopyFile(cfgPath, path.Join(configdir.LocalConfig(AppName), backupCfgName))
		}
	}
	a.Config = cfg
}

func (a *App) initMPV() error {
	p := player.NewWithClientName(AppName)
	c := a.Config.LocalPlayback
	c.InMemoryCacheSizeMB = clamp(c.InMemoryCacheSizeMB, 10, 500)
	if err := p.Init(c.AudioExclusive, c.InMemoryCacheSizeMB); err != nil {
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
	a.Config.LocalPlayback.Volume = a.Player.GetVolume()
	a.cancel()
	a.Player.Destroy()
	a.Config.WriteConfigFile(configPath())
}

func configPath() string {
	return path.Join(configdir.LocalConfig(AppName), configFile)
}

func clamp(i, min, max int) int {
	if i < min {
		i = min
	} else if i > max {
		i = max
	}
	return i
}
