package widgets

import (
	"image"

	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Shows the current album art, track name, artist name, and album name
// for the currently playing track. Placed into the left side of the BottomPanel.
type NowPlayingCard struct {
	widget.BaseWidget

	trackName  *widget.Hyperlink
	artistName *MultiHyperlink
	albumName  *widget.Hyperlink
	cover      *TappableImage
	menu       *widget.PopUpMenu

	OnShowCoverImage func()
	OnSetRating      func(rating int)
	OnSetFavorite    func(favorite bool)
	OnAddToPlaylist  func()

	c fyne.CanvasObject
}

func NewNowPlayingCard() *NowPlayingCard {
	n := &NowPlayingCard{
		trackName:  widget.NewHyperlink("", nil),
		artistName: NewMultiHyperlink(),
		albumName:  widget.NewHyperlink("", nil),
	}
	n.ExtendBaseWidget(n)
	n.cover = NewTappableImage(n.onShowCoverImage)
	n.cover.OnTappedSecondary = n.showMenu
	n.trackName.Hidden = true
	n.artistName.Hidden = true
	n.albumName.Hidden = true
	n.albumName.Wrapping = fyne.TextTruncate
	n.trackName.Wrapping = fyne.TextTruncate
	n.trackName.TextStyle.Bold = true
	n.cover.SetMinSize(fyne.NewSize(85, 85))
	n.cover.FillMode = canvas.ImageFillContain

	n.c = container.New(&layouts.MaxPadLayout{PadLeft: -5},
		container.NewBorder(nil, nil, n.cover, nil,
			container.New(&layouts.MaxPadLayout{PadBottom: -3},
				container.New(&layouts.VboxCustomPadding{ExtraPad: -13}, n.trackName, n.artistName, n.albumName))),
	)
	return n
}

func (n *NowPlayingCard) onShowCoverImage(*fyne.PointEvent) {
	if n.OnShowCoverImage != nil {
		n.OnShowCoverImage()
	}
}

func (n *NowPlayingCard) onSetFavorite(fav bool) {
	if n.OnSetFavorite != nil {
		n.OnSetFavorite(fav)
	}
}

func (n *NowPlayingCard) onSetRating(rating int) {
	if n.OnSetRating != nil {
		n.OnSetRating(rating)
	}
}

func (n *NowPlayingCard) onAddToPlaylist() {
	if n.OnAddToPlaylist != nil {
		n.OnAddToPlaylist()
	}
}

func (n *NowPlayingCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.c)
}

func (n *NowPlayingCard) Update(track, artist string, artistNavigable bool, album string, cover image.Image) {
	n.trackName.SetText(track)
	n.trackName.Hidden = track == ""
	n.artistName.Segments = []MultiHyperlinkSegment{{Text: artist}}
	n.artistName.Hidden = artist == ""
	n.albumName.SetText(album)
	n.albumName.Hidden = album == ""
	n.cover.Image.Image = cover
	n.c.Refresh()
}

func (n *NowPlayingCard) OnArtistNameTapped(f func()) {
	//n.artistName.OnTapped = f
}

func (n *NowPlayingCard) OnAlbumNameTapped(f func()) {
	n.albumName.OnTapped = f
}

func (n *NowPlayingCard) OnTrackNameTapped(f func()) {
	n.trackName.OnTapped = f
}

func (n *NowPlayingCard) showMenu(e *fyne.PointEvent) {
	if n.menu == nil {
		ratingMenu := util.NewRatingSubmenu(n.onSetRating)
		m := fyne.NewMenu("",
			fyne.NewMenuItem("Set favorite", func() { n.onSetFavorite(true) }),
			fyne.NewMenuItem("Unset favorite", func() { n.onSetFavorite(false) }),
			ratingMenu,
			fyne.NewMenuItem("Add to playlist...", func() { n.onAddToPlaylist() }))
		n.menu = widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(n))
	}
	n.menu.ShowAtPosition(e.AbsolutePosition)
}
