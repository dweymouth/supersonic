package ui

import (
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type BottomPanel struct {
	widget.BaseWidget

	imageLoader util.ThumbnailLoader

	NowPlaying  *widgets.NowPlayingCard
	Controls    *widgets.PlayerControls
	AuxControls *widgets.AuxControls

	container *fyne.Container
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(pm *backend.PlaybackManager, im *backend.ImageManager, contr *controller.Controller) *BottomPanel {
	bp := &BottomPanel{}
	bp.ExtendBaseWidget(bp)

	pm.OnSongChange(bp.onSongChange)
	pm.OnPlayTimeUpdate(func(cur, total float64, _ bool) {
		if !pm.IsSeeking() {
			bp.Controls.UpdatePlayTime(cur, total)
		}
	})

	pm.OnPaused(func() {
		bp.Controls.SetPlaying(false)
	})
	pm.OnPlaying(func() {
		bp.Controls.SetPlaying(true)
	})
	pm.OnStopped(func() {
		bp.Controls.SetPlaying(false)
	})

	bp.NowPlaying = widgets.NewNowPlayingCard()
	bp.NowPlaying.OnCoverTapped = func() {
		contr.NavigateTo(controller.NowPlayingRoute(""))
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
	bp.NowPlaying.OnAlbumNameTapped = func() {
		if tr, ok := pm.NowPlaying().(*mediaprovider.Track); ok {
			contr.NavigateTo(controller.AlbumRoute(tr.AlbumID))
		}
	}
	bp.NowPlaying.OnArtistNameTapped = func(artistID string) {
		contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	bp.NowPlaying.OnTrackNameTapped = func() {
		contr.NavigateTo(controller.NowPlayingRoute(pm.NowPlaying().Metadata().ID))
	}
	bp.Controls = widgets.NewPlayerControls()
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

	bp.AuxControls = widgets.NewAuxControls(pm.Volume())
	pm.OnLoopModeChange(bp.AuxControls.SetLoopMode)
	pm.OnVolumeChange(bp.AuxControls.VolumeControl.SetVolume)
	bp.AuxControls.VolumeControl.OnSetVolume = func(v int) {
		_ = pm.SetVolume(v)
	}
	bp.AuxControls.OnChangeLoopMode(func() {
		pm.SetNextLoopMode()
	})

	bp.imageLoader = util.NewThumbnailLoader(im, bp.NowPlaying.SetImage)

	bp.container = container.New(layouts.NewLeftMiddleRightLayout(500),
		bp.NowPlaying, bp.Controls, bp.AuxControls)
	return bp
}

func (bp *BottomPanel) onSongChange(song mediaprovider.MediaItem, _ *mediaprovider.Track) {
	if song == nil {
		bp.NowPlaying.Update(nil)
		bp.imageLoader.Load("")
	} else {
		bp.NowPlaying.Update(song)
		bp.imageLoader.Load(song.Metadata().CoverArtID)
	}
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	bp.ExtendBaseWidget(bp)
	return widget.NewSimpleRenderer(bp.container)
}
