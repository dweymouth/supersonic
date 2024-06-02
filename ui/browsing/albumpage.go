package browsing

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AlbumPage struct {
	widget.BaseWidget

	albumPageState

	disposed     bool
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
	pool    *util.WidgetPool
	mp      mediaprovider.MediaProvider
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	contr   *controller.Controller
}

func NewAlbumPage(
	albumID string,
	cfg *backend.AlbumPageConfig,
	pool *util.WidgetPool,
	pm *backend.PlaybackManager,
	mp mediaprovider.MediaProvider,
	im *backend.ImageManager,
	contr *controller.Controller,
) *AlbumPage {
	return newAlbumPage(albumID, cfg, pool, pm, mp, im, contr, widgets.TracklistSort{})
}

func newAlbumPage(
	albumID string,
	cfg *backend.AlbumPageConfig,
	pool *util.WidgetPool,
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
			pool:    pool,
			pm:      pm,
			mp:      mp,
			im:      im,
			contr:   contr,
		},
	}
	a.ExtendBaseWidget(a)
	if h := pool.Obtain(util.WidgetTypeAlbumPageHeader); h != nil {
		a.header = h.(*AlbumPageHeader)
		a.header.page = a
		a.header.Clear()
	} else {
		a.header = NewAlbumPageHeader(a)
	}
	a.header.page = a
	if t := a.pool.Obtain(util.WidgetTypeCompactTracklist); t != nil {
		a.tracklist = t.(*widgets.Tracklist)
		a.tracklist.Reset()
	} else {
		a.tracklist = widgets.NewTracklist(nil, a.im, true)
	}
	a.tracklist.SetVisibleColumns(a.cfg.TracklistColumns)
	a.tracklist.SetSorting(sort)
	_, canRate := a.mp.(mediaprovider.SupportsRating)
	_, canShare := a.mp.(mediaprovider.SupportsSharing)
	a.tracklist.Options.DisableRating = !canRate
	a.tracklist.Options.DisableSharing = !canShare
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		a.cfg.TracklistColumns = cols
	}
	a.contr.ConnectTracklistActionsWithReplayGainAlbum(a.tracklist)

	a.container = container.NewBorder(
		container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 15, BottomPadding: 10}, a.header),
		nil, nil, nil, container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, BottomPadding: 15}, a.tracklist))

	go a.load()
	return a
}

func (a *AlbumPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPage) Save() SavedPage {
	a.disposed = true
	s := a.albumPageState
	s.sort = a.tracklist.Sorting()
	a.header.page = nil
	a.pool.Release(util.WidgetTypeAlbumPageHeader, a.header)
	a.tracklist.Clear()
	a.pool.Release(util.WidgetTypeCompactTracklist, a.tracklist)
	return &s
}

func (a *AlbumPage) Route() controller.Route {
	return controller.AlbumRoute(a.albumID)
}

var _ CanShowNowPlaying = (*AlbumPage)(nil)

func (a *AlbumPage) OnSongChange(track mediaprovider.MediaItem, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlayingID = sharedutil.MediaItemIDOrEmptyStr(track)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.MediaItemIDOrEmptyStr(lastScrobbledIfAny))
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

var _ Scrollable = (*AlbumPage)(nil)

func (a *AlbumPage) Scroll(scrollAmt float32) {
	a.tracklist.Scroll(scrollAmt)
}

// should be called asynchronously
func (a *AlbumPage) load() {
	album, err := a.mp.GetAlbum(a.albumID)
	if err != nil {
		log.Printf("Failed to get album: %s", err.Error())
		return
	}
	if a.disposed {
		return
	}
	a.header.Update(album, a.im)
	a.tracklist.Options.ShowDiscNumber = len(album.Tracks) > 0 && album.Tracks[0].DiscNumber != album.Tracks[len(album.Tracks)-1].DiscNumber
	a.tracks = album.Tracks
	a.tracklist.SetTracks(album.Tracks)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
}

type AlbumPageHeader struct {
	widget.BaseWidget

	albumID string
	coverID string

	page *AlbumPage

	cover            *widgets.ImagePlaceholder
	titleLabel       *widget.RichText
	releaseTypeLabel *widget.RichText
	artistLabel      *widgets.MultiHyperlink
	artistLabelSpace *util.HSpace // TODO: remove when no longer needed
	genreLabel       *widgets.MultiHyperlink
	miscLabel        *widget.Label
	shareMenuItem    *fyne.MenuItem

	toggleFavButton *widgets.FavoriteButton

	fullSizeCoverFetching bool

	container *fyne.Container
}

func NewAlbumPageHeader(page *AlbumPage) *AlbumPageHeader {
	// due to widget reuse a.page can change so page MUST NOT
	// be directly captured in a closure throughout this function!
	a := &AlbumPageHeader{page: page}
	a.ExtendBaseWidget(a)
	a.cover = widgets.NewImagePlaceholder(myTheme.AlbumIcon, 225)
	a.cover.OnTapped = func(*fyne.PointEvent) { go a.showPopUpCover() }

	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Truncation = fyne.TextTruncateEllipsis
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.releaseTypeLabel = widget.NewRichText(
		&widget.TextSegment{Text: "Album", Style: util.BoldRichTextStyle},
		&widget.TextSegment{Text: " by", Style: widget.RichTextStyle{Inline: true}},
	)
	a.artistLabel = widgets.NewMultiHyperlink()
	a.artistLabel.OnTapped = func(id string) {
		a.page.contr.NavigateTo(controller.ArtistRoute(id))
	}
	a.artistLabelSpace = util.NewHSpace(0) // updated in Update
	a.genreLabel = widgets.NewMultiHyperlink()
	a.genreLabel.OnTapped = func(genre string) {
		a.page.contr.NavigateTo(controller.GenreRoute(genre))
	}
	a.miscLabel = widget.NewLabel("")
	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		go a.page.pm.PlayAlbum(a.page.albumID, 0, false)
	})
	shuffleBtn := widget.NewButtonWithIcon("Shuffle", myTheme.ShuffleIcon, func() {
		a.page.pm.LoadTracks(a.page.tracklist.GetTracks(), backend.Replace, true)
		a.page.pm.PlayFromBeginning()
	})
	var pop *widget.PopUpMenu
	menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuBtn.OnTapped = func() {
		if pop == nil {
			playNext := fyne.NewMenuItem("Play next", func() {
				go a.page.pm.LoadAlbum(a.albumID, backend.InsertNext, false /*shuffle*/)
			})
			playNext.Icon = myTheme.PlayNextIcon
			queue := fyne.NewMenuItem("Add to queue", func() {
				go a.page.pm.LoadAlbum(a.albumID, backend.Append, false /*shuffle*/)
			})
			queue.Icon = theme.ContentAddIcon()
			playlist := fyne.NewMenuItem("Add to playlist...", func() {
				a.page.contr.DoAddTracksToPlaylistWorkflow(
					sharedutil.TracksToIDs(a.page.tracks))
			})
			playlist.Icon = myTheme.PlaylistIcon
			download := fyne.NewMenuItem("Download...", func() {
				a.page.contr.ShowDownloadDialog(a.page.tracks, a.titleLabel.String())
			})
			download.Icon = theme.DownloadIcon()
			info := fyne.NewMenuItem("Show Info...", func() {
				a.page.contr.ShowAlbumInfoDialog(a.albumID, a.titleLabel.String(), a.cover.Image())
			})
			info.Icon = theme.InfoIcon()
			a.shareMenuItem = fyne.NewMenuItem("Share...", func() {
				a.page.contr.ShowShareDialog(a.albumID)
			})
			a.shareMenuItem.Icon = myTheme.ShareIcon
			menu := fyne.NewMenu("", playNext, queue, playlist, download, info, a.shareMenuItem)
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		_, canShare := page.mp.(mediaprovider.SupportsSharing)
		a.shareMenuItem.Disabled = !canShare
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+menuBtn.Size().Height))
	}
	a.toggleFavButton = widgets.NewFavoriteButton(func() { go a.toggleFavorited() })

	// TODO: Create a nicer custom layout to set this up properly
	//   OR once TODO in MultiHyperlink to use RichText as a provider is solved,
	//   extend MultiHyperlink to support prepending rich text segments and
	//   don't use two separate widgets here at all.
	// n.b. cannot place MultiHyperlink in a HBox or it collapses in width
	artistReleaseTypeLine := container.NewStack(
		a.releaseTypeLabel,
		container.NewBorder(nil, nil, a.artistLabelSpace, nil, a.artistLabel))
	// TODO: there's got to be a way to make this less convoluted. Custom layout?
	a.container = util.AddHeaderBackground(
		container.NewBorder(nil, nil, a.cover, nil,
			container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-10),
				a.titleLabel,
				container.NewVBox(
					container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-12), artistReleaseTypeLine, a.genreLabel, a.miscLabel),
					container.NewVBox(
						container.NewHBox(util.NewHSpace(2), playButton, shuffleBtn, menuBtn),
						container.NewHBox(util.NewHSpace(2), a.toggleFavButton),
					),
				),
			),
		))
	return a
}

func (a *AlbumPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *AlbumPageHeader) Update(album *mediaprovider.AlbumWithTracks, im *backend.ImageManager) {
	a.albumID = album.ID
	a.coverID = album.CoverArtID
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = album.Name
	a.releaseTypeLabel.Segments[0].(*widget.TextSegment).Text = util.DisplayReleaseType(album.ReleaseTypes)
	a.releaseTypeLabel.Refresh() // needed so MinSize returns correct width below
	a.artistLabelSpace.Width = a.releaseTypeLabel.MinSize().Width - 16
	a.artistLabel.BuildSegments(album.ArtistNames, album.ArtistIDs)
	a.genreLabel.BuildSegments(album.Genres, album.Genres)
	a.miscLabel.SetText(formatMiscLabelStr(album))
	a.toggleFavButton.IsFavorited = album.Favorite
	a.Refresh()

	go func() {
		if cover, err := im.GetCoverThumbnail(album.CoverArtID); err == nil {
			a.cover.SetImage(cover, true)
			a.cover.Refresh()
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
}

func (a *AlbumPageHeader) Clear() {
	a.albumID = ""
	a.coverID = ""
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = ""
	a.artistLabel.Segments = nil
	a.genreLabel.Segments = nil
	a.miscLabel.SetText("")
	a.toggleFavButton.IsFavorited = false
	a.fullSizeCoverFetching = false
	a.cover.SetImage(nil, false)
	a.cover.Refresh()
}

func (a *AlbumPageHeader) toggleFavorited() {
	params := mediaprovider.RatingFavoriteParameters{AlbumIDs: []string{a.albumID}}
	a.page.mp.SetFavorite(params, a.toggleFavButton.IsFavorited)
}

func (a *AlbumPageHeader) showPopUpCover() {
	if a.fullSizeCoverFetching {
		return
	}
	a.fullSizeCoverFetching = true
	defer func() { a.fullSizeCoverFetching = false }()
	cover, err := a.page.im.GetFullSizeCoverArt(a.coverID)
	if err != nil {
		log.Printf("error getting full size album cover: %s", err.Error())
		return
	}
	if a.page != nil {
		a.page.contr.ShowPopUpImage(cover)
	}
}

func formatMiscLabelStr(a *mediaprovider.AlbumWithTracks) string {
	var discs string
	if len(a.Tracks) > 0 {
		if discCount := a.Tracks[len(a.Tracks)-1].DiscNumber; discCount > 1 {
			discs = fmt.Sprintf("%d discs · ", discCount)
		}
	}
	tracks := "tracks"
	if a.TrackCount == 1 {
		tracks = "track"
	}
	yearStr := strconv.Itoa(a.Year)
	if a.ReissueYear > a.Year {
		yearStr += fmt.Sprintf(" (reissued %d)", a.ReissueYear)
	}
	return fmt.Sprintf("%s · %d %s · %s%s", yearStr, a.TrackCount, tracks, discs, util.SecondsToTimeString(float64(a.Duration)))
}

func (s *albumPageState) Restore() Page {
	return newAlbumPage(s.albumID, s.cfg, s.pool, s.pm, s.mp, s.im, s.contr, s.sort)
}
