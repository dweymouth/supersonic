package ui

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type BottomPanel struct {
	widget.BaseWidget

	ImageManager    *backend.ImageManager
	playbackManager *backend.PlaybackManager

	NowPlaying  *widgets.NowPlayingCard
	Controls    *widgets.PlayerControls
	AuxControls *widgets.AuxControls

	coverArtID string
	container  *fyne.Container
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(p *player.Player, pm *backend.PlaybackManager, contr *controller.Controller) *BottomPanel {
	bp := &BottomPanel{playbackManager: pm}
	bp.ExtendBaseWidget(bp)

	bp.playbackManager.OnSongChange(bp.onSongChange)
	bp.playbackManager.OnPlayTimeUpdate(func(cur, total float64) {
		if !bp.playbackManager.IsSeeking() {
			bp.Controls.UpdatePlayTime(cur, total)
		}
	})

	p.OnPaused(func() {
		bp.Controls.SetPlaying(false)
	})
	p.OnPlaying(func() {
		bp.Controls.SetPlaying(true)
	})
	p.OnStopped(func() {
		bp.Controls.SetPlaying(false)
	})
	pm.OnLoopModeChange(func(mode string) {
		bp.Controls.SetLoopMode(mode)
	})

	bp.NowPlaying = widgets.NewNowPlayingCard()
	bp.NowPlaying.OnShowCoverImage = func() {
		im, err := bp.ImageManager.GetFullSizeCoverArt(bp.coverArtID)
		if err != nil {
			log.Printf("error getting full size cover image: %s", err.Error())
		} else {
			contr.ShowPopUpImage(im)
		}
	}
	bp.NowPlaying.OnSetFavorite = func(fav bool) {
		contr.SetTrackFavorites([]string{bp.playbackManager.NowPlaying().ID}, fav)
	}
	bp.NowPlaying.OnSetRating = func(rating int) {
		contr.SetTrackRatings([]string{bp.playbackManager.NowPlaying().ID}, rating)
	}
	bp.NowPlaying.OnAddToPlaylist = func() {
		contr.DoAddTracksToPlaylistWorkflow([]string{bp.playbackManager.NowPlaying().ID})
	}
	bp.NowPlaying.OnAlbumNameTapped(func() {
		contr.NavigateTo(controller.AlbumRoute(bp.playbackManager.NowPlaying().AlbumID))
	})
	bp.NowPlaying.OnArtistNameTapped(func() {
		contr.NavigateTo(controller.ArtistRoute(bp.playbackManager.NowPlaying().ArtistIDs[0]))
	})
	bp.NowPlaying.OnTrackNameTapped(func() {
		contr.NavigateTo(controller.NowPlayingRoute(bp.playbackManager.NowPlaying().ID))
	})
	bp.Controls = widgets.NewPlayerControls()
	bp.Controls.OnPlayPause(func() {
		p.PlayPause()
	})
	bp.Controls.OnSeekNext(func() {
		p.SeekNext()
	})
	bp.Controls.OnSeekPrevious(func() {
		p.SeekBackOrPrevious()
	})
	bp.Controls.OnSeek(func(f float64) {
		p.Seek(fmt.Sprintf("%d", int(f*100)), player.SeekAbsolutePercent)
	})
	bp.Controls.OnChangeLoopMode(func() {
		bp.playbackManager.SetNextLoopMode()
	})

	bp.AuxControls = widgets.NewAuxControls(p.GetVolume())
	bp.AuxControls.VolumeControl.OnVolumeChanged = func(v int) {
		_ = p.SetVolume(v)
	}

	bp.container = container.New(layouts.NewLeftMiddleRightLayout(500),
		bp.NowPlaying, bp.Controls, bp.AuxControls)
	return bp
}

func (bp *BottomPanel) onSongChange(song, _ *mediaprovider.Track) {
	if song == nil {
		bp.NowPlaying.Update("", "", false, "", nil)
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
		bp.NowPlaying.Update(song.Name, song.ArtistNames[0], song.ArtistIDs[0] != "", song.Album, im)
	}
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	bp.ExtendBaseWidget(bp)
	return widget.NewSimpleRenderer(bp.container)
}
