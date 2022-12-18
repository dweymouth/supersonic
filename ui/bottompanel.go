package ui

import (
	"gomuse/backend"
	"gomuse/player"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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

func NewBottomPanel(p *player.Player, pm *backend.PlaybackManager, im *backend.ImageManager) fyne.CanvasObject {
	n := NewNowPlayingCard()
	c := NewPlayerControls(p, pm)

	pm.OnSongChange(func(song *subsonic.Child) {
		if song == nil {
			n.Update("", "", "", nil)
		} else {
			im, _ := im.GetAlbumThumbnail(song.AlbumID)
			n.Update(song.Title, song.Artist, song.Album, im)
		}
	})

	return container.New(newBottomPanelLayout(500, n, c, nil), n, c)
}
