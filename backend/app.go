package backend

import (
	"context"
	"debug/pe"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/dweymouth/supersonic/backend/ipc"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/backend/util"
	"github.com/google/uuid"

	"github.com/20after4/configdir"
	"github.com/zalando/go-keyring"
)

const (
	configFile     = "config.toml"
	portableDir    = "supersonic_portable"
	savedQueueFile = "saved_queue.json"
	themesDir      = "themes"
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
	WinSMTC         *SMTC
	ipcServer       ipc.IPCServer

	// UI callbacks to be set in main
	OnReactivate func()
	OnExit       func()

	appName        string
	displayAppName string
	appVersionTag  string
	configDir      string
	cacheDir       string
	portableMode   bool

	isFirstLaunch bool // set by config file reader
	bgrndCtx      context.Context
	cancel        context.CancelFunc

	lastWrittenCfg Config

	logFile *os.File
}

func (a *App) VersionTag() string {
	return a.appVersionTag
}

func StartupApp(appName, displayAppName, appVersion, appVersionTag, latestReleaseURL string) (*App, error) {
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

	var logFile *os.File
	if isWindowsGUI() {
		// Can't log to console in Windows GUI app so log to file instead
		if f, err := os.Create(filepath.Join(confDir, "supersonic.log")); err == nil {
			log.SetOutput(f)
			logFile = f
		}
	}

	a := &App{
		logFile:        logFile,
		appName:        appName,
		displayAppName: displayAppName,
		appVersionTag:  appVersionTag,
		configDir:      confDir,
		cacheDir:       cacheDir,
		portableMode:   portableMode,
	}
	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())
	a.readConfig()

	cli, _ := ipc.Connect()
	if HaveCommandLineOptions() {
		if err := a.checkFlagsAndSendIPCMsg(cli); err != nil {
			// we were supposed to control another instance and couldn't
			log.Fatalf("error sending IPC message: %s", err.Error())
		}
		return nil, ErrAnotherInstance
	} else if cli != nil && !a.Config.Application.AllowMultiInstance {
		log.Println("Another instance is running. Reactivating it...")
		cli.Show()
		return nil, ErrAnotherInstance
	}

	log.Printf("Starting %s...", appName)
	log.Printf("Using config dir: %s", confDir)
	log.Printf("Using cache dir: %s", cacheDir)

	a.UpdateChecker = NewUpdateChecker(appVersionTag, latestReleaseURL, &a.Config.Application.LastCheckedVersion)
	a.UpdateChecker.Start(a.bgrndCtx, 24*time.Hour)

	if err := a.initMPV(); err != nil {
		return nil, err
	}
	if err := a.setupMPV(); err != nil {
		return nil, err
	}

	a.ServerManager = NewServerManager(appName, appVersion, a.Config, !portableMode /*use keyring*/)
	a.PlaybackManager = NewPlaybackManager(a.bgrndCtx, a.ServerManager, a.LocalPlayer, &a.Config.Playback, &a.Config.Scrobbling, &a.Config.Transcoding, &a.Config.Application)
	a.ImageManager = NewImageManager(a.bgrndCtx, a.ServerManager, cacheDir)
	a.Config.Application.MaxImageCacheSizeMB = clamp(a.Config.Application.MaxImageCacheSizeMB, 1, 500)
	a.ImageManager.SetMaxOnDiskCacheSizeBytes(int64(a.Config.Application.MaxImageCacheSizeMB) * 1_048_576)
	a.ServerManager.SetPrefetchAlbumCoverCallback(func(coverID string) {
		_, _ = a.ImageManager.GetCoverThumbnail(coverID)
	})

	a.PlaybackManager.OnPlaying(func() {
		SetSystemSleepDisabled(true)
	})
	a.PlaybackManager.OnPaused(func() {
		SetSystemSleepDisabled(false)
	})
	a.PlaybackManager.OnStopped(func() {
		SetSystemSleepDisabled(false)
	})

	// Start IPC server if another not already running in a different instance
	if cli == nil {
		ipc.DestroyConn() // cleanup socket possibly orphaned by crashed process
		listener, err := ipc.Listen()
		if err == nil {
			a.ipcServer = ipc.NewServer(a.PlaybackManager, a.callOnReactivate,
				func() { _ = a.callOnExit() })
			go a.ipcServer.Serve(listener)
		} else {
			log.Printf("error starting IPC server: %s", err.Error())
		}
	}

	// OS media center integrations
	// Linux MPRIS
	a.setupMPRIS(displayAppName)
	// MacOS MPNowPlayingInfoCenter
	InitMPMediaHandler(a.PlaybackManager, func(id string) (string, error) {
		a.ImageManager.GetCoverThumbnail(id) // ensure image is cached locally
		return a.ImageManager.GetCoverArtUrl(id)
	})
	// Windows SMTC is initialized from main once we have a window HWND.

	a.startConfigWriter(a.bgrndCtx)

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

func (a *App) callOnExit() error {
	if a.OnExit == nil {
		return errors.New("no quit handler registered")
	}
	go func() {
		time.Sleep(10 * time.Millisecond)
		a.OnExit()
	}()
	return nil
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
	a.MPRISHandler.OnQuit = a.callOnExit
	a.MPRISHandler.Start()
}

func (a *App) SetupWindowsSMTC(hwnd uintptr) {
	smtc, err := InitSMTCForWindow(hwnd)
	if err != nil {
		log.Printf("error initializing SMTC: %d", err)
		return
	}
	a.WinSMTC = smtc
	smtc.UpdateMetadata(a.displayAppName, "")

	smtc.OnButtonPressed(func(btn SMTCButton) {
		switch btn {
		case SMTCButtonPlay:
			a.PlaybackManager.Continue()
		case SMTCButtonPause:
			a.PlaybackManager.Pause()
		case SMTCButtonNext:
			a.PlaybackManager.SeekNext()
		case SMTCButtonPrevious:
			a.PlaybackManager.SeekBackOrPrevious()
		case SMTCButtonStop:
			a.PlaybackManager.Stop()
		}
	})
	smtc.OnSeek(func(millis int) {
		a.PlaybackManager.SeekSeconds(float64(millis) / 1000)
	})

	a.PlaybackManager.OnSongChange(func(nowPlaying mediaprovider.MediaItem, _ *mediaprovider.Track) {
		if nowPlaying == nil {
			smtc.UpdateMetadata("Supersonic", "")
			return
		}
		meta := nowPlaying.Metadata()
		smtc.UpdateMetadata(meta.Name, strings.Join(meta.Artists, ", "))
		smtc.UpdatePosition(0, meta.Duration*1000)
		go func() {
			a.ImageManager.GetCoverThumbnail(meta.CoverArtID) // ensure image is cached locally
			if path, err := a.ImageManager.GetCoverArtPath(meta.CoverArtID); err == nil {
				smtc.SetThumbnail(path)
			}
		}()
	})
	a.PlaybackManager.OnSeek(func() {
		dur := a.PlaybackManager.NowPlaying().Metadata().Duration
		smtc.UpdatePosition(int(a.PlaybackManager.CurrentPlayer().GetStatus().TimePos*1000), dur*1000)
	})
	a.PlaybackManager.OnPlaying(func() {
		smtc.SetEnabled(true)
		smtc.UpdatePlaybackState(SMTCPlaybackStatePlaying)
	})
	a.PlaybackManager.OnPaused(func() {
		smtc.SetEnabled(true)
		smtc.UpdatePlaybackState(SMTCPlaybackStatePaused)
	})
	a.PlaybackManager.OnStopped(func() {
		smtc.SetEnabled(false)
		smtc.UpdatePlaybackState(SMTCPlaybackStateStopped)
	})
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
	if a.logFile != nil {
		a.logFile.Close()
	}
	repeatMode := "None"
	switch a.PlaybackManager.GetLoopMode() {
	case LoopOne:
		repeatMode = "One"
	case LoopAll:
		repeatMode = "All"
	}
	a.Config.Playback.RepeatMode = repeatMode
	a.Config.Playback.Autoplay = a.PlaybackManager.IsAutoplay()
	a.Config.LocalPlayback.Volume = a.LocalPlayer.GetVolume()
	a.SavePlayQueueIfEnabled()
	a.SaveConfigFile()

	if a.ipcServer != nil {
		a.ipcServer.Shutdown(a.bgrndCtx)
	}
	a.MPRISHandler.Shutdown()
	if a.WinSMTC != nil {
		a.WinSMTC.Shutdown()
	}
	a.PlaybackManager.DisableCallbacks()
	a.PlaybackManager.Stop() // will trigger scrobble check
	a.cancel()
	a.LocalPlayer.Destroy()
}

func (a *App) SavePlayQueueIfEnabled() {
	if !a.Config.Application.SavePlayQueue {
		return
	}
	var queueServer mediaprovider.CanSavePlayQueue = nil
	if a.Config.Application.SaveQueueToServer {
		if qs, ok := a.ServerManager.Server.(mediaprovider.CanSavePlayQueue); ok {
			queueServer = qs
		}
	}
	SavePlayQueue(a.ServerManager.ServerID.String(), a.PlaybackManager, path.Join(a.configDir, savedQueueFile), queueServer)
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

	a.PlaybackManager.LoadTracks(queue.Tracks, Replace, false)
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

func (a *App) checkFlagsAndSendIPCMsg(cli *ipc.Client) error {
	if cli == nil {
		return errors.New("no IPC connection")
	}
	switch {
	case *FlagPlay:
		return cli.Play()
	case *FlagPause:
		return cli.Pause()
	case *FlagPlayPause:
		return cli.PlayPause()
	case *FlagPrevious:
		return cli.SeekBackOrPrevious()
	case *FlagNext:
		return cli.SeekNext()
	case VolumeCLIArg >= 0:
		return cli.SetVolume(VolumeCLIArg)
	case VolumePctCLIArg != 0:
		return cli.AdjustVolumePct(VolumePctCLIArg)
	case SeekToCLIArg >= 0:
		return cli.SeekSeconds(SeekToCLIArg)
	case SeekByCLIArg != 0:
		return cli.SeekBySeconds(SeekByCLIArg)
	default:
		return nil
	}
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

func isWindowsGUI() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// check executable for windows GUI flag
	// https://stackoverflow.com/questions/58813512/is-it-possible-to-detect-if-go-binary-was-compiled-with-h-windowsgui-at-runtime
	fileName, err := os.Executable()
	if err != nil {
		return false
	}
	fl, err := pe.Open(fileName)
	if err != nil {
		return false
	}
	defer fl.Close()

	var subsystem uint16
	if header, ok := fl.OptionalHeader.(*pe.OptionalHeader64); ok {
		subsystem = header.Subsystem
	} else if header, ok := fl.OptionalHeader.(*pe.OptionalHeader32); ok {
		subsystem = header.Subsystem
	}

	return subsystem == 2 /*IMAGE_SUBSYSTEM_WINDOWS_GUI*/
}
