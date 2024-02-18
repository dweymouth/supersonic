package ui

import (
	"image"
	"time"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type BottomPanel struct {
	widget.BaseWidget

	ImageManager *backend.ImageManager

	NowPlaying  *widgets.NowPlayingCard
	Controls    *widgets.PlayerControls
	AuxControls *widgets.AuxControls

	coverArtID string
	container  *fyne.Container
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(pm *backend.PlaybackManager, contr *controller.Controller) *BottomPanel {
	bp := &BottomPanel{}
	bp.ExtendBaseWidget(bp)

	pm.OnSongChange(bp.onSongChange)
	pm.OnPlayTimeUpdate(func(cur, total float64) {
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
		contr.NavigateTo(controller.FullscreenRoute())
	}
	bp.NowPlaying.OnSetFavorite = func(fav bool) {
		contr.SetTrackFavorites([]string{pm.NowPlaying().ID}, fav)
	}
	bp.NowPlaying.OnSetRating = func(rating int) {
		contr.SetTrackRatings([]string{pm.NowPlaying().ID}, rating)
	}
	bp.NowPlaying.OnAddToPlaylist = func() {
		contr.DoAddTracksToPlaylistWorkflow([]string{pm.NowPlaying().ID})
	}
	bp.NowPlaying.OnAlbumNameTapped = func() {
		contr.NavigateTo(controller.AlbumRoute(pm.NowPlaying().AlbumID))
	}
	bp.NowPlaying.OnArtistNameTapped = func(artistID string) {
		contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	bp.NowPlaying.OnTrackNameTapped = func() {
		contr.NavigateTo(controller.NowPlayingRoute(pm.NowPlaying().ID))
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

	bp.container = container.New(layouts.NewLeftMiddleRightLayout(500),
		bp.NowPlaying, bp.Controls, bp.AuxControls)
	return bp
}

func (bp *BottomPanel) onSongChange(song, _ *mediaprovider.Track) {
	if song == nil {
		bp.NowPlaying.Update("", []string{}, []string{}, "", nil)
	} else {
		bp.coverArtID = song.CoverArtID
		var im image.Image
		if bp.ImageManager != nil {
			// set image to expire not long after the length of the song
			// if song is played through without much pausing, image will still
			// be in cache for the next song if it's from the same album, or
			// if the user navigates to the album page for the track
			imgTTLSec := song.Duration + 30
			im, _ = bp.ImageManager.GetCoverThumbnailWithTTL(song.CoverArtID, time.Duration(imgTTLSec)*time.Second)
		}
		bp.NowPlaying.Update(song.Name, song.ArtistNames, song.ArtistIDs, song.Album, im)
	}
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	bp.ExtendBaseWidget(bp)
	return widget.NewSimpleRenderer(bp.container)
}
