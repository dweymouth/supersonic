package browsing

import (
	"fmt"
	"log"
	"supersonic/backend"
	"supersonic/sharedutil"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	myTheme "supersonic/ui/theme"
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
	cfg     *backend.AlbumPageConfig
	lm      *backend.LibraryManager
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	sm      *backend.ServerManager
	contr   *controller.Controller
}

func NewAlbumPage(
	albumID string,
	cfg *backend.AlbumPageConfig,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	lm *backend.LibraryManager,
	im *backend.ImageManager,
	contr *controller.Controller,
) *AlbumPage {
	a := &AlbumPage{
		albumPageState: albumPageState{
			albumID: albumID,
			cfg:     cfg,
			sm:      sm,
			pm:      pm,
			lm:      lm,
			im:      im,
			contr:   contr,
		},
	}
	a.ExtendBaseWidget(a)
	a.header = NewAlbumPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns(a.cfg.TracklistColumns)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		a.cfg.TracklistColumns = cols
	}
	a.contr.ConnectTracklistActions(a.tracklist)

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

func (a *AlbumPage) Route() controller.Route {
	return controller.AlbumRoute(a.albumID)
}

func (a *AlbumPage) OnSongChange(song *subsonic.Child, lastScrobbledIfAny *subsonic.Child) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
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

// should be called asynchronously
func (a *AlbumPage) load() {
	album, err := a.lm.GetAlbum(a.albumID)
	if err != nil {
		log.Printf("Failed to get album: %s", err.Error())
		return
	}
	a.header.Update(album, a.im)
	a.tracklist.ShowDiscNumber = album.Song[0].DiscNumber != album.Song[len(album.Song)-1].DiscNumber
	a.tracklist.Tracks = album.Song
	a.tracklist.SetNowPlaying(a.nowPlayingID)
}

type AlbumPageHeader struct {
	widget.BaseWidget

	albumID  string
	coverID  string
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

	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Wrapping = fyne.TextTruncate
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistLabel = widgets.NewCustomHyperlink()
	a.artistLabel.OnTapped = func() {
		page.contr.NavigateTo(controller.ArtistRoute(a.artistID))
	}
	a.genreLabel = widgets.NewCustomHyperlink()
	a.genreLabel.OnTapped = func() {
		page.contr.NavigateTo(controller.GenreRoute(a.genre))
	}
	a.miscLabel = widget.NewLabel("")
	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		go page.pm.PlayAlbum(page.albumID, 0)
	})
	shuffleBtn := widget.NewButtonWithIcon(" Shuffle", myTheme.ShuffleIcon, func() {
		page.pm.LoadTracks(page.tracklist.Tracks, false, true)
		page.pm.PlayFromBeginning()
	})
	var pop *widget.PopUpMenu
	menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuBtn.OnTapped = func() {
		if pop == nil {
			menu := fyne.NewMenu("",
				fyne.NewMenuItem("Add to queue", func() {
					a.page.pm.LoadAlbum(a.albumID, true /*append*/, false /*shuffle*/)
				}),
				fyne.NewMenuItem("Add to playlist...", func() {
					a.page.contr.DoAddTracksToPlaylistWorkflow(
						sharedutil.TracksToIDs(a.page.tracklist.Tracks))
				}))
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+menuBtn.Size().Height))
	}
	a.toggleFavButton = widgets.NewFavoriteButton(func() { go a.toggleFavorited() })

	// Todo: there's got to be a way to make this less convoluted. Custom layout?
	a.container = container.NewBorder(nil, nil, a.cover, nil,
		container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
			a.titleLabel,
			container.NewVBox(
				container.New(&layouts.VboxCustomPadding{ExtraPad: -12}, a.artistLabel, a.genreLabel, a.miscLabel),
				container.NewVBox(
					container.NewHBox(util.NewHSpace(2), playButton, shuffleBtn, menuBtn),
					container.NewHBox(util.NewHSpace(2), a.toggleFavButton),
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
	a.coverID = album.CoverArt
	a.artistID = album.ArtistID
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = album.Name
	a.artistLabel.SetText(album.Artist)
	a.genre = album.Genre
	a.genreLabel.SetText(album.Genre)
	a.miscLabel.SetText(formatMiscLabelStr(album))
	a.toggleFavButton.IsFavorited = !album.Starred.IsZero()
	a.Refresh()

	go func() {
		if cover, err := im.GetCoverThumbnail(album.CoverArt); err == nil {
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
	cover, err := a.page.im.GetFullSizeCoverArt(a.coverID)
	if err != nil {
		log.Printf("error getting full size album cover: %s", err.Error())
		return
	}
	a.page.contr.ShowPopUpImage(cover)
}

func formatMiscLabelStr(a *subsonic.AlbumID3) string {
	var discs string
	if discCount := a.Song[len(a.Song)-1].DiscNumber; discCount > 1 {
		discs = fmt.Sprintf("%d discs · ", discCount)
	}
	tracks := "tracks"
	if a.SongCount == 1 {
		tracks = "track"
	}
	return fmt.Sprintf("%d · %d %s · %s%s", a.Year, a.SongCount, tracks, discs, util.SecondsToTimeString(float64(a.Duration)))
}

func (s *albumPageState) Restore() Page {
	return NewAlbumPage(s.albumID, s.cfg, s.sm, s.pm, s.lm, s.im, s.contr)
}
