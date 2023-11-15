package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/dweymouth/supersonic/backend/util"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	"github.com/20after4/configdir"
	"github.com/zalando/go-keyring"
)

const (
	sessionDir          = "session"
	sessionLockFile     = ".lock"
	sessionActivateFile = ".activate"
)

var (
	ErrNoServers       = errors.New("no servers set up")
	ErrAnotherInstance = errors.New("another instance is running")
)

type App struct {
	Config          *Config
	ServerManager   *ServerManager
	ImageManager    *ImageManager
	PlaybackManager *PlaybackManager
	Player          *player.Player
	UpdateChecker   UpdateChecker
	MPRISHandler    *MPRISHandler

	// UI callbacks to be set in main
	OnReactivate func()
	OnExit       func()

	appName       string
	appVersionTag string
	configFile    string

	isFirstLaunch bool // set by config file reader
	bgrndCtx      context.Context
	cancel        context.CancelFunc

	lastWrittenCfg Config
}

func (a *App) VersionTag() string {
	return a.appVersionTag
}

func StartupApp(appName, displayAppName, appVersionTag, configFile, latestReleaseURL string) (*App, error) {
	sessionPath := configdir.LocalConfig(appName, sessionDir)
	if _, err := os.Stat(path.Join(sessionPath, sessionLockFile)); err == nil {
		log.Println("Another instance is running. Reactivating it...")
		reactivateFile := path.Join(sessionPath, sessionActivateFile)
		if f, err := os.Create(reactivateFile); err == nil {
			f.Close()
		}
		time.Sleep(750 * time.Millisecond)
		if _, err := os.Stat(reactivateFile); err == nil {
			log.Println("No other instance responded. Starting as normal...")
			os.RemoveAll(sessionPath)
		} else {
			return nil, ErrAnotherInstance
		}
	}

	log.Printf("Starting %s...", appName)
	log.Printf("Using config dir: %s", configdir.LocalConfig(appName))
	log.Printf("Using cache dir: %s", configdir.LocalCache(appName))

	a := &App{appName: appName, appVersionTag: appVersionTag, configFile: configFile}
	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())
	a.readConfig()
	a.startConfigWriter(a.bgrndCtx)

	if !a.Config.Application.AllowMultiInstance {
		log.Println("Creating session lock file")
		os.MkdirAll(sessionPath, 0770)
		if f, err := os.Create(path.Join(sessionPath, sessionLockFile)); err == nil {
			f.Close()
		} else {
			log.Printf("error creating session file: %s", err.Error())
		}
		a.startSessionWatcher(sessionPath)
	}

	a.UpdateChecker = NewUpdateChecker(appVersionTag, latestReleaseURL, &a.Config.Application.LastCheckedVersion)
	a.UpdateChecker.Start(a.bgrndCtx, 24*time.Hour)

	if err := a.initMPV(); err != nil {
		return nil, err
	}
	if err := a.setupMPV(); err != nil {
		return nil, err
	}

	a.ServerManager = NewServerManager(appName, a.Config)
	a.PlaybackManager = NewPlaybackManager(a.bgrndCtx, a.ServerManager, a.Player, &a.Config.Scrobbling, &a.Config.Transcoding)
	a.ImageManager = NewImageManager(a.bgrndCtx, a.ServerManager, configdir.LocalCache(a.appName))
	a.Config.Application.MaxImageCacheSizeMB = clamp(a.Config.Application.MaxImageCacheSizeMB, 1, 500)
	a.ImageManager.SetMaxOnDiskCacheSizeBytes(int64(a.Config.Application.MaxImageCacheSizeMB) * 1_048_576)
	a.ServerManager.SetPrefetchAlbumCoverCallback(func(coverID string) {
		_, _ = a.ImageManager.GetCoverThumbnail(coverID)
	})

	// OS media center integrations
	a.setupMPRIS(displayAppName)
	InitMPMediaHandler(a.Player, a.PlaybackManager, func(id string) (string, error) {
		a.ImageManager.GetCoverThumbnail(id) // ensure image is cached locally
		return a.ImageManager.GetCoverArtUrl(id)
	})

	return a, nil
}

func (a *App) IsFirstLaunch() bool {
	return a.isFirstLaunch
}

func (a *App) readConfig() {
	configdir.MakePath(configdir.LocalConfig(a.appName))
	cfgPath := a.configPath()
	var cfgExists bool
	if _, err := os.Stat(cfgPath); err == nil {
		cfgExists = true
	}
	a.isFirstLaunch = !cfgExists
	cfg, err := ReadConfigFile(cfgPath, a.appVersionTag)
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = DefaultConfig(a.appVersionTag)
		if cfgExists {
			backupCfgName := fmt.Sprintf("%s.bak", a.configFile)
			log.Printf("Config file may be malformed: copying to %s", backupCfgName)
			_ = util.CopyFile(cfgPath, path.Join(configdir.LocalConfig(a.appName), backupCfgName))
		}
	}
	a.Config = cfg
}

func (a *App) startSessionWatcher(sessionPath string) {
	if sessionWatch, err := fsnotify.NewWatcher(); err == nil {
		sessionWatch.Add(sessionPath)
		go func() {
			for {
				select {
				case <-a.bgrndCtx.Done():
					return
				case <-sessionWatch.Events:
					activatePath := path.Join(sessionPath, sessionActivateFile)
					if _, err := os.Stat(activatePath); err == nil {
						os.Remove(path.Join(sessionPath, sessionActivateFile))
						a.callOnReactivate()
					}
				}
			}
		}()
	}
}

// periodically save config file so abnormal exit won't lose settings
func (a *App) startConfigWriter(ctx context.Context) {
	tick := time.NewTicker(2 * time.Minute)
	go func() {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			if !reflect.DeepEqual(&a.lastWrittenCfg, a.Config) {
				a.Config.WriteConfigFile(a.configPath())
				a.lastWrittenCfg = *a.Config
			}
		}
	}()
}

func (a *App) callOnReactivate() {
	if a.OnReactivate != nil {
		a.OnReactivate()
	}
}

func (a *App) initMPV() error {
	p := player.NewWithClientName(a.appName)
	c := a.Config.LocalPlayback
	c.InMemoryCacheSizeMB = clamp(c.InMemoryCacheSizeMB, 10, 500)
	if err := p.Init(c.InMemoryCacheSizeMB); err != nil {
		return fmt.Errorf("failed to initialize mpv player: %s", err.Error())
	}
	a.Player = p
	return nil
}

func (a *App) setupMPV() error {
	a.Config.LocalPlayback.Volume = clamp(a.Config.LocalPlayback.Volume, 0, 100)
	a.Player.SetVolume(a.Config.LocalPlayback.Volume)

	devs, err := a.Player.ListAudioDevices()
	if err != nil {
		return err
	}

	desiredDevice := a.Config.LocalPlayback.AudioDeviceName
	var desiredDeviceAvailable bool
	for _, dev := range devs {
		if dev.Name == desiredDevice {
			desiredDeviceAvailable = true
			break
		}
	}
	if !desiredDeviceAvailable {
		// The audio device the user has configured is not available.
		// Use the default (autoselect) device but leave the setting unchanged,
		// in case the device is later available on a subsequent run of the app
		// (e.g. a USB audio device that is currently unplugged)
		desiredDevice = "auto"
	}
	a.Player.SetAudioDevice(desiredDevice)

	rgainOpts := []string{ReplayGainNone, ReplayGainAlbum, ReplayGainTrack, ReplayGainAuto}
	if !sharedutil.SliceContains(rgainOpts, a.Config.ReplayGain.Mode) {
		a.Config.ReplayGain.Mode = ReplayGainNone
	}
	mode := player.ReplayGainMode(a.Config.ReplayGain.Mode)
	if a.Config.ReplayGain.Mode == ReplayGainAuto {
		mode = player.ReplayGainTrack
	}
	a.Player.SetReplayGainOptions(player.ReplayGainOptions{
		Mode:            mode,
		PreventClipping: a.Config.ReplayGain.PreventClipping,
		PreampGain:      a.Config.ReplayGain.PreampGainDB,
	})
	a.Player.SetAudioExclusive(a.Config.LocalPlayback.AudioExclusive)

	eq := &player.ISO15BandEqualizer{
		EQPreamp: a.Config.LocalPlayback.EqualizerPreamp,
		Disabled: !a.Config.LocalPlayback.EqualizerEnabled,
	}
	copy(eq.BandGains[:], a.Config.LocalPlayback.GraphicEqualizerBands)
	a.Player.SetEqualizer(eq)

	return nil
}

func (a *App) setupMPRIS(mprisAppName string) {
	a.MPRISHandler = NewMPRISHandler(mprisAppName, a.Player, a.PlaybackManager)
	a.MPRISHandler.ArtURLLookup = a.ImageManager.GetCoverArtUrl
	a.MPRISHandler.OnRaise = func() error { a.callOnReactivate(); return nil }
	a.MPRISHandler.OnQuit = func() error {
		if a.OnExit == nil {
			return errors.New("no quit handler registered")
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			a.OnExit()
		}()
		return nil
	}
	a.MPRISHandler.Start()
}

func (a *App) LoginToDefaultServer(string) error {
	serverCfg := a.ServerManager.GetDefaultServer()
	if serverCfg == nil {
		return ErrNoServers
	}
	pass, err := keyring.Get(a.appName, serverCfg.ID.String())
	if err != nil {
		return fmt.Errorf("error reading keyring credentials: %v", err)
	}
	return a.ServerManager.ConnectToServer(serverCfg, pass)
}

func (a *App) DeleteServerCacheDir(serverID uuid.UUID) error {
	path := path.Join(configdir.LocalCache(a.appName), serverID.String())
	log.Printf("Deleting server cache dir: %s", path)
	return os.RemoveAll(path)
}

func (a *App) Shutdown() {
	a.MPRISHandler.Shutdown()
	a.PlaybackManager.DisableCallbacks()
	a.Player.Stop() // will trigger scrobble check
	a.Config.LocalPlayback.Volume = a.Player.GetVolume()
	a.cancel()
	a.Player.Destroy()
	a.Config.WriteConfigFile(a.configPath())
	os.RemoveAll(configdir.LocalConfig(a.appName, sessionDir))
}

func (a *App) SaveConfigFile() {
	a.Config.WriteConfigFile(a.configPath())
	a.lastWrittenCfg = *a.Config
}

func (a *App) configPath() string {
	return path.Join(configdir.LocalConfig(a.appName), a.configFile)
}

func clamp(i, min, max int) int {
	if i < min {
		i = min
	} else if i > max {
		i = max
	}
	return i
}
