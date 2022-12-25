package ui

import (
	"fmt"
	"image"
	"supersonic/backend"
	"supersonic/player"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type bottomPanelLayout struct {
	left, middle, right fyne.CanvasObject
	middleWidth         float32
	hbox                fyne.Layout
}

func newBottomPanelLayout(midWidth float32, left, middle, right fyne.CanvasObject) *bottomPanelLayout {
	return &bottomPanelLayout{
		left:        left,
		middle:      middle,
		right:       right,
		middleWidth: midWidth,
		hbox:        layout.NewHBoxLayout(),
	}
}

func (b *bottomPanelLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	hboxSize := b.hbox.MinSize(objects)
	return fyne.Size{
		Height: hboxSize.Height,
		Width:  hboxSize.Width + fyne.Max(0, b.middleWidth-b.middle.MinSize().Width),
	}
}

func (b *bottomPanelLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	midW := fyne.Max(b.middleWidth, b.middle.MinSize().Width)
	lrW := (size.Width - midW) / 2
	b.left.Resize(fyne.NewSize(lrW, size.Height))
	b.left.Move(fyne.NewPos(0, 0))
	b.middle.Resize(fyne.NewSize(midW, size.Height))
	b.middle.Move(fyne.NewPos(lrW+theme.Padding(), 0))
	if b.right != nil {
		b.right.Resize(fyne.NewSize(lrW, size.Height))
		b.right.Move(fyne.NewPos(lrW+theme.Padding()*2, 0))
	}
}

type BottomPanel struct {
	widget.BaseWidget

	ImageManager *backend.ImageManager

	playbackManager *backend.PlaybackManager

	nowPlaying *widgets.NowPlayingCard
	controls   *widgets.PlayerControls
	container  *fyne.Container
}

var _ fyne.Widget = (*BottomPanel)(nil)

func NewBottomPanel(p *player.Player) *BottomPanel {
	bp := &BottomPanel{}
	bp.ExtendBaseWidget(bp)
	p.OnPaused(func() {
		bp.controls.SetPlaying(false)
	})
	p.OnPlaying(func() {
		bp.controls.SetPlaying(true)
	})
	p.OnStopped(func() {
		bp.controls.SetPlaying(false)
	})

	bp.nowPlaying = widgets.NewNowPlayingCard()
	bp.controls = widgets.NewPlayerControls()
	bp.controls.OnPlayPause(func() {
		p.PlayPause()
	})
	bp.controls.OnSeekNext(func() {
		p.SeekNext()
	})
	bp.controls.OnSeekPrevious(func() {
		p.SeekBackOrPrevious()
	})
	bp.controls.OnSeek(func(f float64) {
		p.Seek(fmt.Sprintf("%d", int(f*100)), player.SeekAbsolutePercent)
	})

	bp.container = container.New(newBottomPanelLayout(500, bp.nowPlaying, bp.controls, nil), bp.nowPlaying, bp.controls)
	return bp
}

func (bp *BottomPanel) SetPlaybackManager(pm *backend.PlaybackManager) {
	bp.playbackManager = pm
	pm.OnSongChange(bp.onSongChange)
	pm.OnPlayTimeUpdate(func(cur, total float64) {
		if !pm.IsSeeking() {
			bp.controls.UpdatePlayTime(cur, total)
		}
	})
}

func (bp *BottomPanel) onSongChange(song *subsonic.Child) {
	if song == nil {
		bp.nowPlaying.Update("", "", "", nil)
	} else {
		var im image.Image
		if bp.ImageManager != nil {
			im, _ = bp.ImageManager.GetAlbumThumbnail(song.AlbumID)
		}
		bp.nowPlaying.Update(song.Title, song.Artist, song.Album, im)
	}
}

func (bp *BottomPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(bp.container)
}
