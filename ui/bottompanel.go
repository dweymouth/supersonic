package ui

import (
	"fmt"
	"image"
	"supersonic/backend"
	"supersonic/player"
	"supersonic/ui/browsing"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type BottomPanel struct {
	widget.BaseWidget

	ImageManager    *backend.ImageManager
	playbackManager *backend.PlaybackManager

	NowPlaying  *widgets.NowPlayingCard
	Controls    *widgets.PlayerControls
	AuxControls *widgets.AuxControls
	container   *fyne.Container
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(p *player.Player, nav func(browsing.Route)) *BottomPanel {
	bp := &BottomPanel{}
	bp.ExtendBaseWidget(bp)
	p.OnPaused(func() {
		bp.Controls.SetPlaying(false)
	})
	p.OnPlaying(func() {
		bp.Controls.SetPlaying(true)
	})
	p.OnStopped(func() {
		bp.Controls.SetPlaying(false)
	})

	bp.NowPlaying = widgets.NewNowPlayingCard()
	bp.NowPlaying.OnAlbumNameTapped(func() {
		nav(browsing.AlbumRoute(bp.playbackManager.NowPlaying().AlbumID))
	})
	bp.NowPlaying.OnArtistNameTapped(func() {
		nav(browsing.ArtistRoute(bp.playbackManager.NowPlaying().ArtistID))
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

	bp.AuxControls = widgets.NewAuxControls(p.GetVolume())
	bp.AuxControls.VolumeControl.OnVolumeChanged = func(v int) {
		_ = p.SetVolume(v)
	}

	bp.container = container.New(layouts.NewLeftMiddleRightLayout(500),
		bp.NowPlaying, bp.Controls, bp.AuxControls)
	return bp
}

func (bp *BottomPanel) SetPlaybackManager(pm *backend.PlaybackManager) {
	bp.playbackManager = pm
	pm.OnSongChange(bp.onSongChange)
	pm.OnPlayTimeUpdate(func(cur, total float64) {
		if !pm.IsSeeking() {
			bp.Controls.UpdatePlayTime(cur, total)
		}
	})
}

func (bp *BottomPanel) onSongChange(song *subsonic.Child, _ *subsonic.Child) {
	if song == nil {
		bp.NowPlaying.Update("", "", "", nil)
	} else {
		var im image.Image
		if bp.ImageManager != nil {
			// set image to expire not long after the length of the song
			// if song is played through without much pausing, image will still
			// be in cache for the next song if it's from the same album, or
			// if the user navigates to the album page for the track
			imgTTLSec := song.Duration + 30
			im, _ = bp.ImageManager.GetAlbumThumbnailWithTTL(song.AlbumID, time.Duration(imgTTLSec)*time.Second)
		}
		bp.NowPlaying.Update(song.Title, song.Artist, song.Album, im)
	}
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	bp.ExtendBaseWidget(bp)
	return widget.NewSimpleRenderer(bp.container)
}
