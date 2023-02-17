package browsing

import (
	"fmt"
	"log"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type AlbumPage struct {
	widget.BaseWidget

	albumPageState

	header       *AlbumPageHeader
	tracklist    *widgets.Tracklist
	nowPlayingID string
	container    *fyne.Container
}

type albumPageState struct {
	albumID string
	lm      *backend.LibraryManager
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	sm      *backend.ServerManager
	contr   controller.Controller
	nav     func(Route)
}

func NewAlbumPage(
	albumID string,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	lm *backend.LibraryManager,
	im *backend.ImageManager,
	contr controller.Controller,
	nav func(Route),
) *AlbumPage {
	a := &AlbumPage{
		albumPageState: albumPageState{
			albumID: albumID,
			sm:      sm,
			pm:      pm,
			lm:      lm,
			im:      im,
			nav:     nav,
			contr:   contr,
		},
	}
	a.ExtendBaseWidget(a)
	a.header = NewAlbumPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns([]widgets.TracklistColumn{
		widgets.ColumnArtist, widgets.ColumnTime, widgets.ColumnPlays})
	// connect tracklist actions
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.tracklist.OnAddToQueue = func(tracks []*subsonic.Child) { a.pm.LoadTracks(tracks, true, false) }
	a.tracklist.OnPlaySelection = func(tracks []*subsonic.Child) {
		a.pm.LoadTracks(tracks, false, false)
		a.pm.PlayFromBeginning()
	}
	a.tracklist.OnAddToPlaylist = a.contr.DoAddTracksToPlaylistWorkflow

	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadBottom: 15}, a.tracklist))

	go a.load()
	return a
}

func (a *AlbumPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPage) Save() SavedPage {
	s := a.albumPageState
	return &s
}

func (a *AlbumPage) Route() Route {
	return AlbumRoute(a.albumID)
}

func (a *AlbumPage) OnSongChange(song *subsonic.Child, lastScrobbledIfAny *subsonic.Child) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(lastScrobbledIfAny)
}

func (a *AlbumPage) Reload() {
	go a.load()
}

func (a *AlbumPage) Tapped(*fyne.PointEvent) {
	a.tracklist.UnselectAll()
}

func (a *AlbumPage) SelectAll() {
	a.tracklist.SelectAll()
}

func (a *AlbumPage) onPlayTrackAt(tracknum int) {
	a.pm.LoadTracks(a.tracklist.Tracks, false, false)
	a.pm.PlayTrackAt(tracknum)
}

// should be called asynchronously
func (a *AlbumPage) load() {
	album, err := a.lm.GetAlbum(a.albumID)
	if err != nil {
		log.Printf("Failed to get album: %s", err.Error())
		return
	}
	a.header.Update(album, a.im)
	a.tracklist.Tracks = album.Song
	a.tracklist.SetNowPlaying(a.nowPlayingID)
}

type AlbumPageHeader struct {
	widget.BaseWidget

	albumID  string
	artistID string
	genre    string

	page *AlbumPage

	cover       *widgets.TappableImage
	titleLabel  *widget.RichText
	artistLabel *widgets.CustomHyperlink
	genreLabel  *widgets.CustomHyperlink
	miscLabel   *widget.Label

	toggleFavButton *widgets.FavoriteButton

	container *fyne.Container
}

func NewAlbumPageHeader(page *AlbumPage) *AlbumPageHeader {
	a := &AlbumPageHeader{page: page}
	a.ExtendBaseWidget(a)
	a.cover = widgets.NewTappableImage(func() { go a.showPopUpCover() })
	a.cover.FillMode = canvas.ImageFillContain
	a.cover.SetMinSize(fyne.NewSize(225, 225))
	// due to cache warming we can probably immediately set the cover
	// and not have to set it asynchronously in the Update function
	if im, ok := page.im.GetAlbumThumbnailFromCache(page.albumID); ok {
		a.cover.Image.Image = im
	}
	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Wrapping = fyne.TextTruncate
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistLabel = widgets.NewCustomHyperlink()
	a.artistLabel.OnTapped = func() {
		page.nav(ArtistRoute(a.artistID))
	}
	a.genreLabel = widgets.NewCustomHyperlink()
	a.genreLabel.OnTapped = func() {
		page.nav(GenreRoute(a.genre))
	}
	a.miscLabel = widget.NewLabel("")
	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		page.onPlayTrackAt(0)
	})
	shuffleBtn := widget.NewButtonWithIcon(" Shuffle", res.ResShuffleInvertSvg, func() {
		page.pm.LoadAlbum(page.albumID, false, true)
		page.pm.PlayFromBeginning()
	})
	a.toggleFavButton = widgets.NewFavoriteButton(a.toggleFavorited)

	// Todo: there's got to be a way to make this less convoluted. Custom layout?
	a.container = container.NewBorder(nil, nil, a.cover, nil,
		container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
			a.titleLabel,
			container.NewVBox(
				container.New(&layouts.VboxCustomPadding{ExtraPad: -12}, a.artistLabel, a.genreLabel, a.miscLabel),
				container.NewVBox(
					container.NewHBox(widgets.NewHSpace(2), playButton, shuffleBtn),
					container.NewHBox(widgets.NewHSpace(2), a.toggleFavButton),
				),
			),
		),
	)
	return a
}

func (a *AlbumPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPageHeader) Update(album *subsonic.AlbumID3, im *backend.ImageManager) {
	a.albumID = album.ID
	a.artistID = album.ArtistID
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = album.Name
	a.artistLabel.SetText(album.Artist)
	a.genre = album.Genre
	a.genreLabel.SetText(album.Genre)
	a.miscLabel.SetText(formatMiscLabelStr(album))
	a.toggleFavButton.IsFavorited = !album.Starred.IsZero()
	a.Refresh()

	// cover image was already loaded from cache in consructor
	if a.albumID == album.ID && a.cover.Image.Image != nil {
		return
	}
	go func() {
		if cover, err := im.GetAlbumThumbnail(album.ID); err == nil {
			a.cover.Image.Image = cover
			a.cover.Refresh()
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
}

func (a *AlbumPageHeader) toggleFavorited() {
	if a.toggleFavButton.IsFavorited {
		a.page.sm.Server.Star(subsonic.StarParameters{AlbumIDs: []string{a.albumID}})
	} else {
		a.page.sm.Server.Unstar(subsonic.StarParameters{AlbumIDs: []string{a.albumID}})
	}
}

func (a *AlbumPageHeader) showPopUpCover() {
	cover, err := a.page.im.GetFullSizeAlbumCover(a.albumID)
	if err != nil {
		log.Printf("error getting full size album cover: %s", err.Error())
		return
	}
	a.page.contr.ShowPopUpImage(cover)
}

func formatMiscLabelStr(a *subsonic.AlbumID3) string {
	return fmt.Sprintf("%d · %d tracks · %s", a.Year, a.SongCount, util.SecondsToTimeString(float64(a.Duration)))
}

func (s *albumPageState) Restore() Page {
	return NewAlbumPage(s.albumID, s.sm, s.pm, s.lm, s.im, s.contr, s.nav)
}
