package browsing

import (
	"log"
	"sort"
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
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// these strings should be keys in the translation dictionaries
var (
	discographySorts = []string{"Year (ascending)", "Year (descending)", "Name (A-Z)"}
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
	sortButton   *widgets.IconButton
	nowPlayingID string
	header       *ArtistPageHeader
	container    *fyne.Container
}

const (
	viewTopTracks   = "Top Tracks"
	viewDiscography = "Discography"
)

func NewArtistPage(artistID string, cfg *backend.ArtistPageConfig, pool *util.WidgetPool, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager, contr *controller.Controller) *ArtistPage {
	activeView := 0
	if cfg.InitialView == viewTopTracks {
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
		a.header.Clear()
	} else {
		a.header = NewArtistPageHeader(a)
	}
	a.header.artistPage = a
	if img, ok := im.GetCachedArtistImage(artistID); ok {
		a.header.artistImage.SetImage(img, true /*tappable*/)
	}
	viewToggle := widgets.NewToggleText(0, []string{lang.L("Discography"), lang.L("Top Tracks")})
	viewToggle.SetActivatedLabel(a.activeView)
	viewToggle.OnChanged = a.onViewChange
	a.sortButton = widgets.NewIconButton(myTheme.SortIcon, a.showAlbumSortMenu)
	viewToggleRow := container.NewBorder(nil, nil,
		container.NewHBox(util.NewHSpace(5), viewToggle),
		container.NewHBox(a.sortButton, util.NewHSpace(10)),
		layout.NewSpacer(),
	)
	a.container = container.NewBorder(
		container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 15, BottomPadding: 10}, a.header),
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
	a.header.artistPage = nil
	a.pool.Release(util.WidgetTypeArtistPageHeader, a.header)
	if a.albumGrid != nil {
		a.albumGrid.Clear()
		a.pool.Release(util.WidgetTypeGridView, a.albumGrid)
	}
	return &s
}

var _ CanShowNowPlaying = (*ArtistPage)(nil)

func (a *ArtistPage) OnSongChange(track mediaprovider.MediaItem, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlayingID = sharedutil.MediaItemIDOrEmptyStr(track)
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		tl.SetNowPlaying(a.nowPlayingID)
		tl.IncrementPlayCount(sharedutil.MediaItemIDOrEmptyStr(lastScrobbledIfAny))
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

func (a *ArtistPage) showAlbumSortMenu() {
	m := fyne.NewMenu("")
	oneChecked := false
	for i, s := range util.LocalizeSlice(discographySorts) {
		_i := i
		item := fyne.NewMenuItem(s, func() {
			a.cfg.DiscographySort = discographySorts[_i]
			a.showAlbumGrid(true /*reSort*/)
		})
		if discographySorts[i] == a.cfg.DiscographySort {
			item.Checked = true
			oneChecked = true
		}
		m.Items = append(m.Items, item)
	}
	if !oneChecked {
		m.Items[0].Checked = true
	}
	btnPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a.sortButton)
	btnSize := a.sortButton.Size()
	pop := widget.NewPopUpMenu(m, fyne.CurrentApp().Driver().CanvasForObject(a))
	menuW := pop.MinSize().Width
	pop.ShowAtPosition(fyne.NewPos(btnPos.X+btnSize.Width-menuW, btnPos.Y+btnSize.Height))
}

func (a *ArtistPage) getGridViewAlbumsModel() []widgets.GridViewItemModel {
	if a.artistInfo == nil {
		return nil
	}
	sortFunc := func(x, y int) bool {
		return a.artistInfo.Albums[x].Year < a.artistInfo.Albums[y].Year
	}
	switch a.cfg.DiscographySort {
	case discographySorts[1]: /*year descending*/
		sortFunc = func(x, y int) bool {
			return a.artistInfo.Albums[x].Year > a.artistInfo.Albums[y].Year
		}
	case discographySorts[2]: /*name*/
		sortFunc = func(x, y int) bool {
			return a.artistInfo.Albums[x].Name < a.artistInfo.Albums[y].Name
		}
	}

	sort.Slice(a.artistInfo.Albums, sortFunc)
	return sharedutil.MapSlice(a.artistInfo.Albums, func(al *mediaprovider.Album) widgets.GridViewItemModel {
		return widgets.GridViewItemModel{
			Name:       al.Name,
			ID:         al.ID,
			CoverArtID: al.CoverArtID,
			Secondary:  []string{strconv.Itoa(al.Year)},
		}
	})
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
		a.showAlbumGrid(false /*reSort*/)
	} else {
		a.showTopTracks()
	}
	info, err := a.mp.GetArtistInfo(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist info: %s", err.Error())
	}
	if !a.disposed {
		a.header.UpdateInfo(info)
	}
}

func (a *ArtistPage) showAlbumGrid(reSort bool) {
	if a.albumGrid == nil {
		if a.artistInfo == nil {
			// page not loaded yet or invalid artist
			a.activeView = 0 // if page still loading, will show discography view first
			return
		}
		model := a.getGridViewAlbumsModel()
		if g := a.pool.Obtain(util.WidgetTypeGridView); g != nil {
			a.albumGrid = g.(*widgets.GridView)
			a.albumGrid.Placeholder = myTheme.AlbumIcon
			a.albumGrid.ResetFixed(model)
		} else {
			a.albumGrid = widgets.NewFixedGridView(model, a.im, myTheme.AlbumIcon)
		}
		a.contr.ConnectAlbumGridActions(a.albumGrid)
	} else if reSort {
		model := a.getGridViewAlbumsModel()
		a.albumGrid.ResetFixed(model)
	}
	a.sortButton.Show()
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
			tl = widgets.NewTracklist(ts, a.im, false)
		}
		tl.Options = widgets.TracklistOptions{AutoNumber: true}
		_, canRate := a.mp.(mediaprovider.SupportsRating)
		_, canShare := a.mp.(mediaprovider.SupportsSharing)
		tl.Options.DisableRating = !canRate
		tl.Options.DisableSharing = !canShare
		tl.SetVisibleColumns(a.cfg.TracklistColumns)
		tl.SetSorting(a.trackSort)
		tl.OnVisibleColumnsChanged = func(cols []string) {
			a.cfg.TracklistColumns = cols
		}
		tl.SetNowPlaying(a.nowPlayingID)
		a.contr.ConnectTracklistActions(tl)
		a.tracklistCtr = container.New(
			&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, BottomPadding: 10},
			tl)
	}
	a.sortButton.Hide()
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.tracklistCtr
	a.container.Objects[0].Refresh()
}

func (a *ArtistPage) onViewChange(num int) {
	if num == 0 {
		a.showAlbumGrid(false /*reSort*/)
	} else {
		// needs to request info from server if first time,
		// so call it asynchronously
		go a.showTopTracks()
	}
	a.activeView = num
	if num == 1 {
		a.cfg.InitialView = viewTopTracks
	} else {
		a.cfg.InitialView = viewDiscography
	}
}

func (a *ArtistPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

func (s *artistPageState) Restore() Page {
	return newArtistPage(s.artistID, s.cfg, s.pool, s.pm, s.mp, s.im, s.contr, s.activeView, s.trackSort)
}

const artistBioNotAvailableKey = "Artist biography not available."

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
	//shareMenuItem  *fyne.MenuItem
}

func NewArtistPageHeader(page *ArtistPage) *ArtistPageHeader {
	// due to widget reuse a.artistPage can change so page MUST NOT
	// be directly captured in a closure throughout this function!
	a := &ArtistPageHeader{
		artistPage:     page,
		titleDisp:      widget.NewRichTextWithText(""),
		biographyDisp:  widgets.NewMaxRowsLabel(5, lang.L(artistBioNotAvailableKey)),
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
	a.playBtn = widget.NewButtonWithIcon(lang.L("Play Discography"), theme.MediaPlayIcon(), func() {
		go a.artistPage.pm.PlayArtistDiscography(a.artistID, false /*shuffle*/)
	})
	a.playRadioBtn = widget.NewButtonWithIcon(lang.L("Play Artist Radio"), myTheme.ShuffleIcon, func() {
		// must not pass playArtistRadio func directly to NewButton
		// because the artistPage bound to this header can change when reused
		a.artistPage.playArtistRadio()
	})

	var pop *widget.PopUpMenu
	a.menuBtn = widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	a.menuBtn.OnTapped = func() {
		if pop == nil {
			shuffleTracks := fyne.NewMenuItem(lang.L("Shuffle tracks"), func() {
				go a.artistPage.pm.PlayArtistDiscography(a.artistID, true /*shuffle*/)
			})
			shuffleTracks.Icon = myTheme.TracksIcon
			shuffleAlbums := fyne.NewMenuItem(lang.L("Shuffle albums"), func() {
				go a.artistPage.pm.ShuffleArtistAlbums(a.artistID)
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
	a.biographyDisp.Text = lang.L(artistBioNotAvailableKey)
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
		a.similarArtists.Add(widget.NewLabel(lang.L("Similar artists") + ":"))
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
				container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-10),
					a.titleDisp, a.biographyDisp, a.similarArtists),
				btnContainer),
		))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
