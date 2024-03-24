package browsing

import (
	"log"
	"strconv"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type artistPageState struct {
	artistID   string
	activeView int
	trackSort  widgets.TracklistSort

	pool  *util.WidgetPool
	cfg   *backend.ArtistPageConfig
	pm    *backend.PlaybackManager
	mp    mediaprovider.MediaProvider
	im    *backend.ImageManager
	contr *controller.Controller
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState
	disposed bool

	artistInfo *mediaprovider.ArtistWithAlbums

	albumGrid    *widgets.GridView
	tracklistCtr *fyne.Container
	nowPlayingID string
	header       *ArtistPageHeader
	container    *fyne.Container
}

func NewArtistPage(artistID string, cfg *backend.ArtistPageConfig, pool *util.WidgetPool, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager, contr *controller.Controller) *ArtistPage {
	activeView := 0
	if cfg.InitialView == "Top Tracks" {
		activeView = 1
	}
	return newArtistPage(artistID, cfg, pool, pm, mp, im, contr, activeView, widgets.TracklistSort{})
}

func newArtistPage(artistID string, cfg *backend.ArtistPageConfig, pool *util.WidgetPool, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager, contr *controller.Controller, activeView int, sort widgets.TracklistSort) *ArtistPage {
	a := &ArtistPage{artistPageState: artistPageState{
		artistID:   artistID,
		cfg:        cfg,
		pool:       pool,
		pm:         pm,
		mp:         mp,
		im:         im,
		contr:      contr,
		activeView: activeView,
		trackSort:  sort,
	}}
	a.ExtendBaseWidget(a)
	if h := a.pool.Obtain(util.WidgetTypeArtistPageHeader); h != nil {
		a.header = h.(*ArtistPageHeader)
		a.header.artistPage = a
		a.header.Clear()
	} else {
		a.header = NewArtistPageHeader(a)
	}
	a.header.artistPage = a
	if img, ok := im.GetCachedArtistImage(artistID); ok {
		a.header.artistImage.SetImage(img, true /*tappable*/)
	}
	viewToggle := widgets.NewToggleText(0, []string{"Discography", "Top Tracks"})
	viewToggle.SetActivatedLabel(a.activeView)
	viewToggle.OnChanged = a.onViewChange
	//line := canvas.NewLine(theme.TextColor())
	viewToggleRow := container.NewBorder(nil, nil,
		container.NewHBox(util.NewHSpace(5), viewToggle), nil,
		layout.NewSpacer(),
	)
	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil,
		container.NewBorder(viewToggleRow, nil, nil, nil, layout.NewSpacer()))
	go a.load()
	return a
}

func (a *ArtistPage) Tapped(*fyne.PointEvent) {
	if a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).UnselectAll()
	}
}

var _ CanSelectAll = (*ArtistPage)(nil)

func (a *ArtistPage) SelectAll() {
	if a.activeView == 1 && a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).SelectAll()
	}
}

func (a *ArtistPage) Route() controller.Route {
	return controller.ArtistRoute(a.artistID)
}

func (a *ArtistPage) Reload() {
	go a.load()
}

func (a *ArtistPage) Save() SavedPage {
	a.disposed = true
	s := a.artistPageState
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		s.trackSort = tl.Sorting()
		tl.Clear()
		a.pool.Release(util.WidgetTypeTracklist, tl)
	}
	a.pool.Release(util.WidgetTypeArtistPageHeader, a.header)
	if a.albumGrid != nil {
		a.albumGrid.Clear()
		a.pool.Release(util.WidgetTypeGridView, a.albumGrid)
	}
	return &s
}

var _ CanShowNowPlaying = (*ArtistPage)(nil)

func (a *ArtistPage) OnSongChange(track, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(track)
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		tl.SetNowPlaying(a.nowPlayingID)
		tl.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
	}
}

var _ Scrollable = (*ArtistPage)(nil)

func (g *ArtistPage) Scroll(scrollAmt float32) {
	if g.activeView == 0 && g.albumGrid != nil {
		g.albumGrid.ScrollToOffset(g.albumGrid.GetScrollOffset() + scrollAmt)
	}
}

func (a *ArtistPage) playArtistRadio() {
	go a.pm.PlaySimilarSongs(a.artistID)
}

// should be called asynchronously
func (a *ArtistPage) load() {
	artist, err := a.mp.GetArtist(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist: %s", err.Error())
		return
	}
	if a.disposed {
		return
	}
	a.artistInfo = artist
	a.header.Update(artist)
	if a.activeView == 0 {
		a.showAlbumGrid()
	} else {
		a.showTopTracks()
	}
	info, err := a.mp.GetArtistInfo(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist info: %s", err.Error())
	}
	a.header.UpdateInfo(info)
}

func (a *ArtistPage) showAlbumGrid() {
	if a.albumGrid == nil {
		if a.artistInfo == nil {
			// page not loaded yet or invalid artist
			a.activeView = 0 // if page still loading, will show discography view first
			return
		}
		model := sharedutil.MapSlice(a.artistInfo.Albums, func(al *mediaprovider.Album) widgets.GridViewItemModel {
			return widgets.GridViewItemModel{
				Name:       al.Name,
				ID:         al.ID,
				CoverArtID: al.CoverArtID,
				Secondary:  []string{strconv.Itoa(al.Year)},
			}
		})
		if g := a.pool.Obtain(util.WidgetTypeGridView); g != nil {
			a.albumGrid = g.(*widgets.GridView)
			a.albumGrid.Placeholder = myTheme.AlbumIcon
			a.albumGrid.ResetFixed(model)
		} else {
			a.albumGrid = widgets.NewFixedGridView(model, a.im, myTheme.AlbumIcon)
		}
		a.contr.ConnectAlbumGridActions(a.albumGrid)
	}
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.albumGrid
	a.container.Objects[0].Refresh()
}

func (a *ArtistPage) showTopTracks() {
	if a.tracklistCtr == nil {
		if a.artistInfo == nil {
			// page not loaded yet or invalid artist
			a.activeView = 1 // if page still loading, will show tracks view first
			return
		}
		ts, err := a.mp.GetTopTracks(a.artistInfo.Artist, 20)
		if err != nil {
			log.Printf("error getting top songs: %s", err.Error())
			return
		}
		if a.disposed {
			return
		}
		var tl *widgets.Tracklist
		if t := a.pool.Obtain(util.WidgetTypeTracklist); t != nil {
			tl = t.(*widgets.Tracklist)
			tl.Reset()
			tl.SetTracks(ts)
		} else {
			tl = widgets.NewTracklist(ts)
		}
		tl.Options = widgets.TracklistOptions{AutoNumber: true}
		_, canRate := a.mp.(mediaprovider.SupportsRating)
		_, canShare := a.mp.(mediaprovider.SupportsSharing)
		_, canSongRadio := a.mp.(mediaprovider.SupportsSongRadio)
		tl.Options.DisableRating = !canRate
		tl.Options.DisableSharing = !canShare
		tl.Options.DisableSongRadio = !canSongRadio
		tl.SetVisibleColumns(a.cfg.TracklistColumns)
		tl.SetSorting(a.trackSort)
		tl.OnVisibleColumnsChanged = func(cols []string) {
			a.cfg.TracklistColumns = cols
		}
		tl.SetNowPlaying(a.nowPlayingID)
		a.contr.ConnectTracklistActions(tl)
		a.tracklistCtr = container.New(
			&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadBottom: 10},
			tl)
	}
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.tracklistCtr
	a.container.Objects[0].Refresh()
}

func (a *ArtistPage) onViewChange(num int) {
	if num == 0 {
		a.showAlbumGrid()
	} else {
		// needs to request info from server if first time,
		// so call it asynchronously
		go a.showTopTracks()
	}
	a.activeView = num
	if num == 1 {
		a.cfg.InitialView = "Top Tracks"
	} else {
		a.cfg.InitialView = "Discography"
	}
}

func (a *ArtistPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

func (s *artistPageState) Restore() Page {
	return newArtistPage(s.artistID, s.cfg, s.pool, s.pm, s.mp, s.im, s.contr, s.activeView, s.trackSort)
}

const artistBioNotAvailableStr = "Artist biography not available."

type ArtistPageHeader struct {
	widget.BaseWidget

	artistID       string
	artistPage     *ArtistPage
	artistImage    *widgets.ImagePlaceholder
	titleDisp      *widget.RichText
	biographyDisp  *widgets.MaxRowsLabel
	similarArtists *fyne.Container
	favoriteBtn    *widgets.FavoriteButton
	playBtn        *widget.Button
	playRadioBtn   *widget.Button
	menuBtn        *widget.Button
	container      *fyne.Container
	shareMenuItem  *fyne.MenuItem
}

func NewArtistPageHeader(page *ArtistPage) *ArtistPageHeader {
	// due to widget reuse a.artistPage can change so page MUST NOT
	// be directly captured in a closure throughout this function!
	a := &ArtistPageHeader{
		artistPage:     page,
		titleDisp:      widget.NewRichTextWithText(""),
		biographyDisp:  widgets.NewMaxRowsLabel(5, artistBioNotAvailableStr),
		similarArtists: container.NewHBox(),
	}
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistImage = widgets.NewImagePlaceholder(myTheme.ArtistIcon, 225)
	a.artistImage.OnTapped = func(*fyne.PointEvent) {
		if im := a.artistImage.Image(); im != nil {
			a.artistPage.contr.ShowPopUpImage(im)
		}
	}
	a.favoriteBtn = widgets.NewFavoriteButton(func() { go a.toggleFavorited() })
	a.playBtn = widget.NewButtonWithIcon("Play Discography", theme.MediaPlayIcon(), func() {
		go a.artistPage.contr.PlayArtistDiscography(a.artistID, false /*shuffle*/)
	})
	a.playRadioBtn = widget.NewButtonWithIcon("Play Artist Radio", myTheme.ShuffleIcon, a.artistPage.playArtistRadio)

	var pop *widget.PopUpMenu
	a.menuBtn = widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	a.menuBtn.OnTapped = func() {
		if pop == nil {
			shuffleTracks := fyne.NewMenuItem("Shuffle tracks", func() {
				go a.artistPage.contr.PlayArtistDiscography(a.artistID, true /*shuffle*/)
			})
			shuffleTracks.Icon = myTheme.TracksIcon
			shuffleAlbums := fyne.NewMenuItem("Shuffle albums", func() {
				go a.artistPage.contr.ShuffleArtistAlbums(a.artistID)
			})
			shuffleAlbums.Icon = myTheme.AlbumIcon
			menu := fyne.NewMenu("", shuffleTracks, shuffleAlbums)
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a.menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+a.menuBtn.Size().Height))
	}

	// TODO: Uncomment and merge into OnTapped above when at least one media provider supports sharing artists.
	// shareMenuItem = fyne.NewMenuItem("Share...", func() {
	// 	a.artistPage.contr.ShowShareDialog(a.artistID)
	// })
	// a.shareMenuItem.Icon = myTheme.ShareIcon
	//
	// canShareArtists := false
	// if r, canShare := a.artistPage.mp.(mediaprovider.SupportsSharing); canShare {
	// 	canShareArtists = r.CanShareArtists()
	// }
	// shareMenuItem.Disabled = !canShareArtists

	a.biographyDisp.Wrapping = fyne.TextWrapWord
	a.biographyDisp.Truncation = fyne.TextTruncateEllipsis
	a.ExtendBaseWidget(a)
	a.createContainer()
	return a
}

func (a *ArtistPageHeader) Clear() {
	a.artistID = ""
	a.favoriteBtn.IsFavorited = false
	a.titleDisp.Segments[0].(*widget.TextSegment).Text = ""
	a.biographyDisp.Text = artistBioNotAvailableStr
	for _, obj := range a.similarArtists.Objects {
		obj.Hide()
	}
	a.artistImage.SetImage(nil, false)
}

func (a *ArtistPageHeader) Update(artist *mediaprovider.ArtistWithAlbums) {
	if artist == nil {
		return
	}
	a.favoriteBtn.IsFavorited = artist.Favorite
	a.favoriteBtn.Refresh()
	a.artistID = artist.ID
	a.titleDisp.Segments[0].(*widget.TextSegment).Text = artist.Name
	a.titleDisp.Refresh()
	if artist.CoverArtID == "" {
		return
	}
	if im, err := a.artistPage.im.GetCoverThumbnail(artist.CoverArtID); err != nil {
		log.Printf("failed to load artist image: %v", err)
	} else {
		a.artistImage.SetImage(im, true /*tappable*/)
	}
}

func (a *ArtistPageHeader) UpdateInfo(info *mediaprovider.ArtistInfo) {
	if info == nil {
		return
	}

	if text := util.PlaintextFromHTMLString(info.Biography); text != "" {
		a.biographyDisp.SetText(text)
	}

	if len(a.similarArtists.Objects) == 0 {
		a.similarArtists.Add(widget.NewLabel("Similar Artists:"))
	}
	for _, obj := range a.similarArtists.Objects {
		obj.Hide()
	}
	for i, art := range info.SimilarArtists {
		if i == 0 {
			a.similarArtists.Objects[0].Show() // "Similar Artists:" label
		}
		if i == 4 {
			break
		}
		if len(a.similarArtists.Objects) <= i+1 {
			a.similarArtists.Add(widget.NewHyperlink("", nil))
		}
		h := a.similarArtists.Objects[i+1].(*widget.Hyperlink)
		h.SetText(art.Name)
		h.OnTapped = func(id string) func() {
			return func() { a.artistPage.contr.NavigateTo(controller.ArtistRoute(id)) }
		}(art.ID)
		h.Show()
	}
	a.similarArtists.Refresh()

	if info.ImageURL != "" {
		if a.artistImage.HaveImage() {
			_ = a.artistPage.im.RefreshCachedArtistImageIfExpired(a.artistID, info.ImageURL)
		} else {
			im, err := a.artistPage.im.FetchAndCacheArtistImage(a.artistID, info.ImageURL)
			if err == nil {
				a.artistImage.SetImage(im, true /*tappable*/)
			}
		}
	}
}

func (a *ArtistPageHeader) toggleFavorited() {
	params := mediaprovider.RatingFavoriteParameters{ArtistIDs: []string{a.artistID}}
	a.artistPage.mp.SetFavorite(params, a.favoriteBtn.IsFavorited)
}

func (a *ArtistPageHeader) createContainer() {
	btnContainer := container.NewHBox(util.NewHSpace(2), a.favoriteBtn, a.playBtn, a.playRadioBtn, a.menuBtn)

	a.container = util.AddHeaderBackground(
		container.NewBorder(nil, nil, a.artistImage, nil,
			container.NewVBox(
				container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
					a.titleDisp, a.biographyDisp, a.similarArtists),
				btnContainer),
		))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
