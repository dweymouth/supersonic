package browsing

import (
	"fmt"
	"log"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AlbumPage struct {
	widget.BaseWidget

	albumPageState

	header       *AlbumPageHeader
	tracks       []*mediaprovider.Track
	tracklist    *widgets.Tracklist
	nowPlayingID string
	container    *fyne.Container
}

type albumPageState struct {
	albumID string
	sort    widgets.TracklistSort
	cfg     *backend.AlbumPageConfig
	mp      mediaprovider.MediaProvider
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	contr   *controller.Controller
}

func NewAlbumPage(
	albumID string,
	cfg *backend.AlbumPageConfig,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
	contr *controller.Controller,
) *AlbumPage {
	return newAlbumPage(albumID, cfg, pm, mp, im, contr, widgets.TracklistSort{})
}

func newAlbumPage(
	albumID string,
	cfg *backend.AlbumPageConfig,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
	contr *controller.Controller,
	sort widgets.TracklistSort,
) *AlbumPage {
	a := &AlbumPage{
		albumPageState: albumPageState{
			albumID: albumID,
			cfg:     cfg,
			pm:      pm,
			mp:      mp,
			im:      im,
			contr:   contr,
		},
	}
	a.ExtendBaseWidget(a)
	a.header = NewAlbumPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns(a.cfg.TracklistColumns)
	a.tracklist.SetSorting(sort)
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
	s.sort = a.tracklist.Sorting()
	return &s
}

func (a *AlbumPage) Route() controller.Route {
	return controller.AlbumRoute(a.albumID)
}

func (a *AlbumPage) OnSongChange(track, lastScrobbledIfAny *mediaprovider.Track) {
	if track == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = track.ID
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
	album, err := a.mp.GetAlbum(a.albumID)
	if err != nil {
		log.Printf("Failed to get album: %s", err.Error())
		return
	}
	a.header.Update(album, a.im)
	a.tracklist.ShowDiscNumber = album.Tracks[0].DiscNumber != album.Tracks[len(album.Tracks)-1].DiscNumber
	a.tracks = album.Tracks
	a.tracklist.SetTracks(album.Tracks)
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
	a.cover = widgets.NewTappableImage(func(*fyne.PointEvent) { go a.showPopUpCover() })
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
		go page.pm.PlayAlbum(page.albumID, 0, false)
	})
	shuffleBtn := widget.NewButtonWithIcon(" Shuffle", myTheme.ShuffleIcon, func() {
		page.pm.LoadTracks(page.tracklist.GetTracks(), false, true)
		page.pm.PlayFromBeginning()
	})
	var pop *widget.PopUpMenu
	menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuBtn.OnTapped = func() {
		if pop == nil {
			menu := fyne.NewMenu("",
				fyne.NewMenuItem("Add to queue", func() {
					go a.page.pm.LoadAlbum(a.albumID, true /*append*/, false /*shuffle*/)
				}),
				fyne.NewMenuItem("Add to playlist...", func() {
					a.page.contr.DoAddTracksToPlaylistWorkflow(
						sharedutil.TracksToIDs(a.page.tracks))
				}),
				fyne.NewMenuItem("Download", func() {
					go func() {
						album, err := a.page.mp.GetAlbum(a.albumID)
						if err != nil {
							log.Print("Error while getting the album: ", album)
						}
						a.page.contr.ShowDownloadDialog(a.page.tracks, album.Name)
					}()
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

func (a *AlbumPageHeader) Update(album *mediaprovider.AlbumWithTracks, im *backend.ImageManager) {
	a.albumID = album.ID
	a.coverID = album.CoverArtID
	a.artistID = album.ArtistIDs[0]
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = album.Name
	a.artistLabel.SetText(album.ArtistNames[0])
	a.genre = album.Genres[0]
	a.genreLabel.SetText(album.Genres[0])
	a.miscLabel.SetText(formatMiscLabelStr(album))
	a.toggleFavButton.IsFavorited = album.Favorite
	a.Refresh()

	go func() {
		if cover, err := im.GetCoverThumbnail(album.CoverArtID); err == nil {
			a.cover.Image.Image = cover
			a.cover.Refresh()
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
}

func (a *AlbumPageHeader) toggleFavorited() {
	params := mediaprovider.RatingFavoriteParameters{AlbumIDs: []string{a.albumID}}
	a.page.mp.SetFavorite(params, a.toggleFavButton.IsFavorited)
}

func (a *AlbumPageHeader) showPopUpCover() {
	cover, err := a.page.im.GetFullSizeCoverArt(a.coverID)
	if err != nil {
		log.Printf("error getting full size album cover: %s", err.Error())
		return
	}
	a.page.contr.ShowPopUpImage(cover)
}

func formatMiscLabelStr(a *mediaprovider.AlbumWithTracks) string {
	var discs string
	if discCount := a.Tracks[len(a.Tracks)-1].DiscNumber; discCount > 1 {
		discs = fmt.Sprintf("%d discs · ", discCount)
	}
	tracks := "tracks"
	if a.TrackCount == 1 {
		tracks = "track"
	}
	return fmt.Sprintf("%d · %d %s · %s%s", a.Year, a.TrackCount, tracks, discs, util.SecondsToTimeString(float64(a.Duration)))
}

func (s *albumPageState) Restore() Page {
	return newAlbumPage(s.albumID, s.cfg, s.pm, s.mp, s.im, s.contr, s.sort)
}
