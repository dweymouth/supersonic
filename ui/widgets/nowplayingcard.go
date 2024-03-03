package widgets

import (
	"image"

	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
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

	DisableRating bool

	trackName  *widget.Hyperlink
	artistName *MultiHyperlink
	albumName  *widget.Hyperlink
	cover      *TappableImage
	menu       *widget.PopUpMenu
	ratingMenu *fyne.MenuItem

	OnTrackNameTapped  func()
	OnArtistNameTapped func(artistID string)
	OnAlbumNameTapped  func()
	OnCoverTapped      func()
	OnSetRating        func(rating int)
	OnSetFavorite      func(favorite bool)
	OnAddToPlaylist    func()
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
	n.albumName.Hidden = true
	n.albumName.Truncation = fyne.TextTruncateEllipsis
	n.trackName.Truncation = fyne.TextTruncateEllipsis
	n.trackName.TextStyle.Bold = true
	n.cover.SetMinSize(fyne.NewSize(85, 85))
	n.cover.FillMode = canvas.ImageFillContain
	n.cover.ScaleMode = canvas.ImageScaleFastest
	n.cover.Hidden = true
	n.albumName.OnTapped = n.onAlbumNameTapped
	n.artistName.OnTapped = n.onArtistNameTapped
	n.trackName.OnTapped = n.onTrackNameTapped

	return n
}

func (n *NowPlayingCard) MinSize() fyne.Size {
	// prop up height for when cover image is hidden
	return fyne.NewSize(n.BaseWidget.MinSize().Width, 85)
}

func (n *NowPlayingCard) onAlbumNameTapped() {
	if n.OnAlbumNameTapped != nil {
		n.OnAlbumNameTapped()
	}
}

func (n *NowPlayingCard) onArtistNameTapped(artistID string) {
	if n.OnArtistNameTapped != nil {
		n.OnArtistNameTapped(artistID)
	}
}

func (n *NowPlayingCard) onTrackNameTapped() {
	if n.OnTrackNameTapped != nil {
		n.OnTrackNameTapped()
	}
}

func (n *NowPlayingCard) onShowCoverImage(*fyne.PointEvent) {
	if n.OnCoverTapped != nil {
		n.OnCoverTapped()
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
	c := container.New(&layouts.MaxPadLayout{PadLeft: -4},
		container.NewBorder(nil, nil, n.cover, nil,
			container.New(&layouts.MaxPadLayout{PadTop: -2},
				container.New(&layouts.VboxCustomPadding{ExtraPad: -13}, n.trackName, n.artistName, n.albumName))),
	)
	return widget.NewSimpleRenderer(c)
}

func (n *NowPlayingCard) Update(track string, artists, artistIDs []string, album string, cover image.Image) {
	n.trackName.SetText(track)
	n.trackName.Hidden = track == ""
	n.artistName.BuildSegments(artists, artistIDs)
	n.albumName.SetText(album)
	n.albumName.Hidden = album == ""
	n.cover.Image.Image = cover
	n.cover.Hidden = cover == nil
	n.Refresh()
}

func (n *NowPlayingCard) showMenu(e *fyne.PointEvent) {
	if n.menu == nil {
		n.ratingMenu = util.NewRatingSubmenu(n.onSetRating)
		favorite := fyne.NewMenuItem("Set favorite", func() { n.onSetFavorite(true) })
		favorite.Icon = myTheme.FavoriteIcon
		unfavorite := fyne.NewMenuItem("Unset favorite", func() { n.onSetFavorite(false) })
		unfavorite.Icon = myTheme.NotFavoriteIcon
		playlist := fyne.NewMenuItem("Add to playlist...", func() { n.onAddToPlaylist() })
		playlist.Icon = myTheme.PlaylistIcon

		m := fyne.NewMenu("", favorite, unfavorite, n.ratingMenu, playlist)
		n.menu = widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(n))
	}
	n.ratingMenu.Disabled = n.DisableRating
	n.menu.ShowAtPosition(e.AbsolutePosition)
}
