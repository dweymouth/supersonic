package browsing

import (
	"fmt"
	"log"
	"supersonic/backend"
	"supersonic/ui/layouts"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type AlbumPage struct {
	widget.BaseWidget

	albumID      string
	im           *backend.ImageManager
	lm           *backend.LibraryManager
	nav          func(Route)
	header       *AlbumPageHeader
	tracklist    *widgets.Tracklist
	nowPlayingID string
	container    *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewAlbumPage(albumID string, lm *backend.LibraryManager, im *backend.ImageManager, nav func(Route)) *AlbumPage {
	a := &AlbumPage{albumID: albumID, lm: lm, im: im, nav: nav}
	a.ExtendBaseWidget(a)
	a.header = NewAlbumPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadBottom: 15}, a.tracklist))
	a.loadAsync()
	return a
}

func (a *AlbumPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *AlbumPage) Save() SavedPage {
	return &savedAlbumPage{
		albumID: a.albumID,
		lm:      a.lm,
		im:      a.im,
		nav:     a.nav,
	}
}

func (a *AlbumPage) Route() Route {
	return AlbumRoute(a.albumID)
}

func (a *AlbumPage) OnSongChange(song *subsonic.Child) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
}

func (a *AlbumPage) Reload() {
	a.loadAsync()
}

func (a *AlbumPage) onPlayTrackAt(tracknum int) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(a.albumID, tracknum)
	}
}

func (a *AlbumPage) loadAsync() {
	go func() {
		album, err := a.lm.GetAlbum(a.albumID)
		if err != nil {
			log.Printf("Failed to get album: %s", err.Error())
			return
		}
		a.header.Update(album, a.im)
		a.tracklist.Tracks = album.Song
		a.tracklist.SetNowPlaying(a.nowPlayingID)
	}()
}

type AlbumPageHeader struct {
	widget.BaseWidget

	albumID  string
	artistID string
	genre    string

	cover       *canvas.Image
	titleLabel  *widget.RichText
	artistLabel *widgets.CustomHyperlink
	genreLabel  *widgets.CustomHyperlink
	miscLabel   *widget.Label

	playButton *widget.Button

	container *fyne.Container
}

func NewAlbumPageHeader(page *AlbumPage) *AlbumPageHeader {
	a := &AlbumPageHeader{}
	a.ExtendBaseWidget(a)
	a.cover = &canvas.Image{FillMode: canvas.ImageFillContain}
	a.cover.SetMinSize(fyne.NewSize(225, 225))
	// due to cache warming we can probably immediately set the cover
	// and not have to set it asynchronously in the Update function
	if im, ok := page.im.GetAlbumThumbnailFromCache(page.albumID); ok {
		a.cover.Image = im
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
	a.playButton = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		page.onPlayTrackAt(0)
	})

	a.container = container.NewBorder(nil, nil, a.cover, nil,
		container.NewVBox(
			a.titleLabel,
			container.New(&layouts.VboxCustomPadding{ExtraPad: -10}, a.artistLabel, a.genreLabel, a.miscLabel),
			container.NewHBox(a.playButton),
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
	a.Refresh()

	// cover image was already loaded from cache in consructor
	if a.albumID == album.ID && a.cover.Image != nil {
		return
	}
	go func() {
		if cover, err := im.GetAlbumThumbnail(album.ID); err == nil {
			a.cover.Image = cover
			a.cover.Refresh()
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
}

func formatMiscLabelStr(a *subsonic.AlbumID3) string {
	return fmt.Sprintf("%d · %d tracks · %s", a.Year, a.SongCount, util.SecondsToTimeString(float64(a.Duration)))
}

type savedAlbumPage struct {
	albumID string
	lm      *backend.LibraryManager
	im      *backend.ImageManager
	nav     func(Route)
}

func (s *savedAlbumPage) Restore() Page {
	return NewAlbumPage(s.albumID, s.lm, s.im, s.nav)
}
