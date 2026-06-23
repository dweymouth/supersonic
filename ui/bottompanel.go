package ui

import (
	"fmt"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type BottomPanel struct {
	widget.BaseWidget

	imageLoader util.ThumbnailLoader

	NowPlaying  *widgets.NowPlayingCard
	Controls    *widgets.PlayerControls
	AuxControls *widgets.AuxControls

	container *fyne.Container

	cfg *backend.Config
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(pm *backend.PlaybackManager, im *backend.ImageManager, contr *controller.Controller, cfg *backend.Config) *BottomPanel {
	bp := &BottomPanel{cfg: cfg}
	bp.ExtendBaseWidget(bp)

	pm.OnSongChange(bp.onSongChange)
	pm.OnRadioMetadataChange(bp.onRadioMetadataChange)
	pm.OnWaveformImgUpdate(bp.updateWaveformImg)
	pm.OnPlayTimeUpdate(func(cur, total float64, _ bool) {
		fyne.Do(func() {
			if !pm.IsSeeking() {
				bp.Controls.UpdatePlayTime(cur, total)
			}
			bp.updateSoftwareVolumeLock(pm)
		})
	})

	pm.OnPaused(util.FyneDoFunc(func() { bp.Controls.SetPlaying(false) }))
	pm.OnPlaying(util.FyneDoFunc(func() { bp.Controls.SetPlaying(true) }))
	pm.OnStopped(util.FyneDoFunc(func() {
		bp.Controls.SetPlaying(false)
		bp.Controls.UpdatePlayTime(0, 0)
		bp.AuxControls.VolumeControl.SetSoftwareVolumeLocked(false, "")
	}))

	bp.NowPlaying = widgets.NewNowPlayingCard()
	bp.NowPlaying.ShowAlbumYear = cfg.AlbumsPage.ShowYears
	bp.NowPlaying.OnCoverTapped = func() {
		contr.NavigateTo(controller.NowPlayingRoute())
	}
	bp.NowPlaying.OnSetFavorite = func(fav bool) {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.SetTrackFavorites([]string{tr.ID}, fav)
		}
	}
	bp.NowPlaying.OnSetRating = func(rating int) {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.SetTrackRatings([]string{tr.ID}, rating)
		}
	}
	bp.NowPlaying.OnAddToPlaylist = func() {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.DoAddTracksToPlaylistWorkflow([]string{tr.ID})
		}
	}
	bp.NowPlaying.OnAlbumNameTapped = func(albumID string) {
		contr.NavigateTo(controller.AlbumRoute(albumID))
	}
	bp.NowPlaying.OnArtistNameTapped = func(artistID string) {
		contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	bp.NowPlaying.OnTrackNameTapped = func() {
		contr.NavigateTo(controller.NowPlayingRoute())
	}
	bp.NowPlaying.OnShowTrackInfo = func() {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.ShowTrackInfoDialog(tr)
		}
	}
	bp.NowPlaying.OnShare = func() {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.ShowShareDialog(tr.ID)
		}
	}
	bp.Controls = widgets.NewPlayerControls(cfg.Playback.UseWaveformSeekbar, pm.GetLoopMode(), pm.IsShuffle())
	bp.Controls.OnPlayPause(func() {
		pm.PlayPause()
	})
	bp.Controls.OnSeekNext(func() {
		pm.SeekNext()
	})
	bp.Controls.OnSeekPrevious(func() {
		pm.SeekBackOrPrevious()
	})
	bp.Controls.OnSeek(func(f float64) {
		pm.SeekFraction(f)
	})
	bp.Controls.OnChangeLoopMode(func() {
		pm.SetNextLoopMode()
	})
	bp.Controls.OnChangeShuffle = func(shuffle bool) {
		pm.SetShuffle(shuffle)
	}
	pm.OnLoopModeChange(func(lm backend.LoopMode) {
		fyne.Do(func() { bp.Controls.SetLoopMode(lm) })
	})
	pm.OnShuffleChange(func(sh bool) {
		fyne.Do(func() { bp.Controls.SetShuffle(sh) })
	})

	bp.AuxControls = widgets.NewAuxControls(pm.Volume(), pm.IsAutoplay())
	pm.OnVolumeChange(func(vol int) {
		fyne.Do(func() { bp.AuxControls.VolumeControl.SetVolume(vol) })
	})
	pm.OnPlayerChange(func() {
		_, local := pm.CurrentPlayer().(player.LocalPlayer)
		fyne.Do(func() {
			bp.AuxControls.SetIsRemotePlayer(!local)
			bp.updateSoftwareVolumeLock(pm)
		})
	})
	bp.AuxControls.VolumeControl.OnSetVolume = func(v int) {
		if bp.AuxControls.VolumeControl.SoftwareVolumeLocked() {
			return
		}
		pm.SetVolume(v)
	}
	bp.AuxControls.OnChangeAutoplay = func(autoplay bool) {
		pm.SetAutoplay(autoplay)
	}
	bp.AuxControls.OnShowPlayQueue(contr.ShowPopUpPlayQueue)
	bp.AuxControls.OnShowCastMenu(contr.ShowCastMenu)

	bp.imageLoader = util.NewThumbnailLoader(im, bp.NowPlaying.SetImage)
	bp.updateSoftwareVolumeLock(pm)

	bp.container = container.New(layouts.NewLeftMiddleRightLayout(300, 0.4),
		bp.NowPlaying, bp.Controls, bp.AuxControls)
	return bp
}

func (bp *BottomPanel) updateSoftwareVolumeLock(pm *backend.PlaybackManager) {
	locked := false
	reason := ""
	quality := widgets.QualityPathInfo{
		Badge:  "Audio",
		Status: lang.L("Stopped"),
	}
	if pm.PlaybackStatus().State != player.Stopped {
		if localPlayer, ok := pm.CurrentPlayer().(player.LocalPlayer); ok {
			if info, err := localPlayer.GetMediaInfo(); err == nil {
				quality = qualityPathInfo(info)
				if info.SoftwareVolumeLocked {
					locked = true
					reason = lang.L("Bit Perfect playback is active; software volume is locked at 100%. Use hardware/DAC volume or turn Bit Perfect off.")
				}
			}
		}
	}
	bp.AuxControls.SetQualityPath(quality)
	bp.AuxControls.VolumeControl.SetSoftwareVolumeLocked(locked, reason)
}

func qualityPathInfo(info player.MediaInfo) widgets.QualityPathInfo {
	status := info.SignalStatus
	if status == "" {
		if info.BitPerfectActive {
			status = "Bit-Perfect"
		} else if info.ExclusiveActive {
			status = "Exclusive Mode"
		} else {
			status = "Shared Output"
		}
	}
	badge := "Audio"
	if info.SourceIsDSD && info.DSDRate > 0 {
		badge = fmt.Sprintf("DSD %.4f MHz", float64(info.DSDRate)/1000000)
		if info.DoPCarrierRate > 0 {
			badge = fmt.Sprintf("%s / DoP %.1f kHz", badge, float64(info.DoPCarrierRate)/1000)
		}
	} else if info.OutputSamplerate > 0 && info.OutputFormat != "" {
		badge = fmt.Sprintf("%.1f kHz / %s", float64(info.OutputSamplerate)/1000, info.OutputFormat)
	}
	if info.BitPerfectActive {
		badge = "Bit-perfect " + badge
	}
	return widgets.QualityPathInfo{
		Badge:                badge,
		Status:               status,
		SourceFormat:         info.SourceFormat,
		DecodePath:           info.DecodePath,
		OutputPath:           info.OutputPath,
		DACFormat:            info.DACFormat,
		DeviceName:           info.DeviceName,
		DeviceTransport:      info.DeviceTransport,
		PlaybackPath:         info.PlaybackPath,
		Reason:               info.BitPerfectReason,
		ExclusiveActive:      info.ExclusiveActive,
		BitPerfectActive:     info.BitPerfectActive,
		OutputMixable:        info.OutputMixable,
		PhysicalFormatCount:  info.DevicePhysicalFormats,
		ExclusiveFormatCount: info.DeviceExclusiveFormats,
		DeviceMinSampleRate:  info.DeviceMinSampleRate,
		DeviceMaxSampleRate:  info.DeviceMaxSampleRate,
		DeviceMaxBitDepth:    info.DeviceMaxBitDepth,
		DeviceChannels:       info.DeviceChannels,
		SourceIsDSD:          info.SourceIsDSD,
		DSDRate:              info.DSDRate,
		DoPCarrierRate:       info.DoPCarrierRate,
	}
}

func (bp *BottomPanel) onSongChange(song mediaprovider.MediaItem, _ *mediaprovider.Track) {
	fyne.Do(func() {
		if song == nil {
			bp.NowPlaying.Update(nil)
			bp.imageLoader.Load("")
		} else {
			bp.NowPlaying.Update(song)
			bp.imageLoader.Load(song.Metadata().CoverArtID)
		}
	})
}

func (bp *BottomPanel) onRadioMetadataChange(radioName, title, artist string) {
	fyne.Do(func() {
		bp.NowPlaying.Update(&mediaprovider.Track{
			Title:       title,
			ArtistNames: []string{artist},
			Album:       radioName,
		})
	})
}

func (bp *BottomPanel) updateWaveformImg(img *backend.WaveformImage) {
	fyne.Do(func() {
		bp.Controls.UpdateWaveformImg(img)
	})
}

func (bp *BottomPanel) Refresh() {
	bp.NowPlaying.ShowAlbumYear = bp.cfg.AlbumsPage.ShowYears
	bp.BaseWidget.Refresh()
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	bp.ExtendBaseWidget(bp)
	return widget.NewSimpleRenderer(bp.container)
}
