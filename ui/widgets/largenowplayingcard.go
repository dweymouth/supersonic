package widgets

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/theme"
)

// Shows the current album art, track name, artist name, and album name
// for the currently playing track. Placed into the left side of the BottomPanel.
type LargeNowPlayingCard struct {
	CaptionedImage

	DisableRating bool

	trackName  *widget.Hyperlink
	artistName *MultiHyperlink
	albumName  *widget.Hyperlink
	cover      *ImagePlaceholder

	OnTrackNameTapped  func()
	OnArtistNameTapped func(artistID string)
	OnAlbumNameTapped  func()
	OnShowCoverImage   func()
	OnSetRating        func(rating int)
	OnSetFavorite      func(favorite bool)
	OnAddToPlaylist    func()
}

func NewLargeNowPlayingCard() *LargeNowPlayingCard {
	n := &LargeNowPlayingCard{
		trackName:  widget.NewHyperlink("", nil),
		artistName: NewMultiHyperlink(),
		albumName:  widget.NewHyperlink("", nil),
		cover:      NewImagePlaceholder(theme.TracksIcon, 300),
	}
	n.ExtendBaseWidget(n)
	// set up the layout
	n.Content = n.cover
	n.Caption = container.New(&layouts.VboxCustomPadding{ExtraPad: -13},
		n.trackName,
		n.albumName,
		n.artistName,
	)

	n.trackName.Hidden = true
	n.albumName.Hidden = true
	n.albumName.Truncation = fyne.TextTruncateEllipsis
	n.trackName.Truncation = fyne.TextTruncateEllipsis
	n.trackName.TextStyle.Bold = true
	n.albumName.OnTapped = n.onAlbumNameTapped
	n.artistName.OnTapped = n.onArtistNameTapped
	n.trackName.OnTapped = n.onTrackNameTapped

	return n
}

func (n *LargeNowPlayingCard) onAlbumNameTapped() {
	if n.OnAlbumNameTapped != nil {
		n.OnAlbumNameTapped()
	}
}

func (n *LargeNowPlayingCard) onArtistNameTapped(artistID string) {
	if n.OnArtistNameTapped != nil {
		n.OnArtistNameTapped(artistID)
	}
}

func (n *LargeNowPlayingCard) onTrackNameTapped() {
	if n.OnTrackNameTapped != nil {
		n.OnTrackNameTapped()
	}
}

func (n *LargeNowPlayingCard) onShowCoverImage(*fyne.PointEvent) {
	if n.OnShowCoverImage != nil {
		n.OnShowCoverImage()
	}
}

func (n *LargeNowPlayingCard) onSetFavorite(fav bool) {
	if n.OnSetFavorite != nil {
		n.OnSetFavorite(fav)
	}
}

func (n *LargeNowPlayingCard) onSetRating(rating int) {
	if n.OnSetRating != nil {
		n.OnSetRating(rating)
	}
}

func (n *LargeNowPlayingCard) Update(track string, artists, artistIDs []string, album string) {
	n.trackName.SetText(track)
	n.trackName.Hidden = track == ""
	n.artistName.BuildSegments(artists, artistIDs)
	n.albumName.SetText(album)
	n.albumName.Hidden = album == ""
	n.Refresh()
}

func (n *LargeNowPlayingCard) SetCoverImage(im image.Image) {
	n.cover.SetImage(im, false)
}
