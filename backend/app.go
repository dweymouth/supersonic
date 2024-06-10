package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"github.com/dweymouth/supersonic/backend/ipc"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	"github.com/20after4/configdir"
	"github.com/zalando/go-keyring"
)

const (
	configFile          = "config.toml"
	portableDir         = "supersonic_portable"
	sessionDir          = "session"
	sessionLockFile     = ".lock"
	sessionActivateFile = ".activate"
	savedQueueFile      = "saved_queue.json"
	themesDir           = "themes"
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
	LocalPlayer     *mpv.Player
	UpdateChecker   UpdateChecker
	MPRISHandler    *MPRISHandler

	// UI callbacks to be set in main
	OnReactivate func()
	OnExit       func()

	appName       string
	appVersionTag string
	configDir     string
	cacheDir      string
	portableMode  bool

	isFirstLaunch bool // set by config file reader
	bgrndCtx      context.Context
	cancel        context.CancelFunc

	lastWrittenCfg Config
}

func (a *App) VersionTag() string {
	return a.appVersionTag
}

func StartupApp(appName, displayAppName, appVersionTag, latestReleaseURL string) (*App, error) {
	var confDir, cacheDir string
	portableMode := false
	if p := checkPortablePath(); p != "" {
		confDir = path.Join(p, "config")
		cacheDir = path.Join(p, "cache")
		portableMode = true
	} else {
		confDir = configdir.LocalConfig(appName)
		cacheDir = configdir.LocalCache(appName)
	}
	// ensure config and cache dirs exist
	configdir.MakePath(confDir)
	configdir.MakePath(cacheDir)

	cli, err := ipc.Connect()
	if err == nil {
		log.Println("Another instance is running. Reactivating it...")
		cli.Show()
		return nil, ErrAnotherInstance
	}

	/*
		sessionPath := path.Join(confDir, sessionDir)
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
	*/

	log.Printf("Starting %s...", appName)
	log.Printf("Using config dir: %s", confDir)
	log.Printf("Using cache dir: %s", cacheDir)

	a := &App{
		appName:       appName,
		appVersionTag: appVersionTag,
		configDir:     confDir,
		cacheDir:      cacheDir,
		portableMode:  portableMode,
	}
	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())
	a.readConfig()
	a.startConfigWriter(a.bgrndCtx)

	/*
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
	*/

	a.UpdateChecker = NewUpdateChecker(appVersionTag, latestReleaseURL, &a.Config.Application.LastCheckedVersion)
	a.UpdateChecker.Start(a.bgrndCtx, 24*time.Hour)

	if err := a.initMPV(); err != nil {
		return nil, err
	}
	if err := a.setupMPV(); err != nil {
		return nil, err
	}

	a.ServerManager = NewServerManager(appName, a.Config, !portableMode /*use keyring*/)
	a.PlaybackManager = NewPlaybackManager(a.bgrndCtx, a.ServerManager, a.LocalPlayer, &a.Config.Scrobbling, &a.Config.Transcoding)
	a.ImageManager = NewImageManager(a.bgrndCtx, a.ServerManager, cacheDir)
	a.Config.Application.MaxImageCacheSizeMB = clamp(a.Config.Application.MaxImageCacheSizeMB, 1, 500)
	a.ImageManager.SetMaxOnDiskCacheSizeBytes(int64(a.Config.Application.MaxImageCacheSizeMB) * 1_048_576)
	a.ServerManager.SetPrefetchAlbumCoverCallback(func(coverID string) {
		_, _ = a.ImageManager.GetCoverThumbnail(coverID)
	})
	listener, _ := ipc.Listen()
	server := ipc.NewServer(a.PlaybackManager, nil)
	go server.Serve(listener)

	// OS media center integrations
	a.setupMPRIS(displayAppName)
	InitMPMediaHandler(a.PlaybackManager, func(id string) (string, error) {
		a.ImageManager.GetCoverThumbnail(id) // ensure image is cached locally
		return a.ImageManager.GetCoverArtUrl(id)
	})

	return a, nil
}

func (a *App) IsFirstLaunch() bool {
	return a.isFirstLaunch
}

func (a *App) IsPortableMode() bool {
	return a.portableMode
}

func (a *App) ThemesDir() string {
	return filepath.Join(a.configDir, themesDir)
}

func checkPortablePath() string {
	if p, err := os.Executable(); err == nil {
		pdirPath := path.Join(filepath.Dir(p), portableDir)
		if s, err := os.Stat(pdirPath); err == nil && s.IsDir() {
			return pdirPath
		}
	}
	return ""
}

func (a *App) readConfig() {
	cfgPath := a.configFilePath()
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
			backupCfgName := fmt.Sprintf("%s.bak", configFile)
			log.Printf("Config file may be malformed: copying to %s", backupCfgName)
			_ = util.CopyFile(cfgPath, path.Join(a.configDir, backupCfgName))
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
				a.Config.WriteConfigFile(a.configFilePath())
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
	p := mpv.NewWithClientName(a.appName)
	c := a.Config.LocalPlayback
	c.InMemoryCacheSizeMB = clamp(c.InMemoryCacheSizeMB, 10, 500)
	if err := p.Init(c.InMemoryCacheSizeMB); err != nil {
		return fmt.Errorf("failed to initialize mpv player: %s", err.Error())
	}
	a.LocalPlayer = p
	return nil
}

func (a *App) setupMPV() error {
	a.Config.LocalPlayback.Volume = clamp(a.Config.LocalPlayback.Volume, 0, 100)
	a.LocalPlayer.SetVolume(a.Config.LocalPlayback.Volume)

	devs, err := a.LocalPlayer.ListAudioDevices()
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
	a.LocalPlayer.SetAudioDevice(desiredDevice)

	rgainOpts := []string{ReplayGainNone, ReplayGainAlbum, ReplayGainTrack, ReplayGainAuto}
	if !slices.Contains(rgainOpts, a.Config.ReplayGain.Mode) {
		a.Config.ReplayGain.Mode = ReplayGainNone
	}
	mode := player.ReplayGainNone
	switch a.Config.ReplayGain.Mode {
	case ReplayGainAlbum:
		mode = player.ReplayGainAlbum
	case ReplayGainTrack:
		mode = player.ReplayGainTrack
	case ReplayGainAuto:
		mode = player.ReplayGainTrack
	}

	a.LocalPlayer.SetReplayGainOptions(player.ReplayGainOptions{
		Mode:            mode,
		PreventClipping: a.Config.ReplayGain.PreventClipping,
		PreampGain:      a.Config.ReplayGain.PreampGainDB,
	})
	a.LocalPlayer.SetAudioExclusive(a.Config.LocalPlayback.AudioExclusive)

	eq := &mpv.ISO15BandEqualizer{
		EQPreamp: a.Config.LocalPlayback.EqualizerPreamp,
		Disabled: !a.Config.LocalPlayback.EqualizerEnabled,
	}
	copy(eq.BandGains[:], a.Config.LocalPlayback.GraphicEqualizerBands)
	a.LocalPlayer.SetEqualizer(eq)

	return nil
}

func (a *App) setupMPRIS(mprisAppName string) {
	a.MPRISHandler = NewMPRISHandler(mprisAppName, a.PlaybackManager)
	a.MPRISHandler.ArtURLLookup = func(id string) (string, error) {
		a.ImageManager.GetCoverThumbnail(id) // ensure image is cached locally
		return a.ImageManager.GetCoverArtUrl(id)
	}
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
	path := path.Join(a.cacheDir, serverID.String())
	log.Printf("Deleting server cache dir: %s", path)
	return os.RemoveAll(path)
}

func (a *App) Shutdown() {
	a.MPRISHandler.Shutdown()
	a.PlaybackManager.DisableCallbacks()
	if a.Config.Application.SavePlayQueue {
		var queueServer mediaprovider.CanSavePlayQueue = nil
		if a.Config.Application.SaveQueueToServer {
			if qs, ok := a.ServerManager.Server.(mediaprovider.CanSavePlayQueue); ok {
				queueServer = qs
			}
		}
		SavePlayQueue(a.ServerManager.ServerID.String(), a.PlaybackManager, path.Join(a.configDir, savedQueueFile), queueServer)
	}
	a.PlaybackManager.Stop() // will trigger scrobble check
	a.Config.LocalPlayback.Volume = a.LocalPlayer.GetVolume()
	a.cancel()
	a.LocalPlayer.Destroy()
	a.Config.WriteConfigFile(a.configFilePath())
	os.RemoveAll(path.Join(a.configDir, sessionDir))
}

func (a *App) LoadSavedPlayQueue() error {
	queueFilePath := path.Join(a.configDir, savedQueueFile)
	queue, err := LoadPlayQueue(queueFilePath, a.ServerManager, a.Config.Application.SaveQueueToServer)
	if err != nil {
		return err
	}
	if len(queue.Tracks) == 0 {
		return nil
	}
	if len(a.PlaybackManager.GetPlayQueue()) > 0 {
		// don't restore play queue if the user has already queued new tracks
		return nil
	}

	if err := a.PlaybackManager.LoadTracks(queue.Tracks, Replace, false); err != nil {
		return err
	}
	if queue.TrackIndex >= 0 && queue.TrackIndex < len(queue.Tracks) {
		// TODO: This isn't ideal but doesn't seem to cause an audible play-for-a-split-second artifact
		a.PlaybackManager.PlayTrackAt(queue.TrackIndex)
		a.PlaybackManager.Pause()
		time.Sleep(100 * time.Millisecond) // MPV seek fails if run quickly after
		a.PlaybackManager.SeekSeconds(queue.TimePos)
	}
	return nil
}

func (a *App) SaveConfigFile() {
	a.Config.WriteConfigFile(a.configFilePath())
	a.lastWrittenCfg = *a.Config
}

func (a *App) configFilePath() string {
	return path.Join(a.configDir, configFile)
}

func clamp(i, min, max int) int {
	if i < min {
		i = min
	} else if i > max {
		i = max
	}
	return i
}
