package browsing

import (
	"log"
	"slices"
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

	gridScrollPos float32 // for album grid (or grouped releases)
	listScrollPos float32 // for Top Tracks list

	sectionVis          widgets.GroupedReleasesSectionVisibility
	sectionVisNeedApply bool
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState
	disposed bool

	artistInfo *mediaprovider.ArtistWithAlbums

	albumGrid       *widgets.GridView
	groupedReleases *widgets.GroupedReleases
	tracklistCtr    *fyne.Container
	sortButton      *widgets.SortChooserButton
	nowPlayingID    string
	header          *ArtistPageHeader
	container       *fyne.Container
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
	return newArtistPage(artistPageState{
		artistID:   artistID,
		cfg:        cfg,
		pool:       pool,
		pm:         pm,
		mp:         mp,
		im:         im,
		contr:      contr,
		activeView: activeView,
	})
}

func newArtistPage(state artistPageState) *ArtistPage {
	a := &ArtistPage{artistPageState: state}
	a.ExtendBaseWidget(a)
	if h := a.pool.Obtain(util.WidgetTypeArtistPageHeader); h != nil {
		a.header = h.(*ArtistPageHeader)
		a.header.Clear()
	} else {
		a.header = NewArtistPageHeader(a)
	}
	a.header.artistPage = a
	if img, ok := state.im.GetCachedArtistImage(state.artistID); ok {
		a.header.artistImage.SetImage(img, true /*tappable*/)
	}
	viewToggle := widgets.NewToggleText(0, []string{lang.L("Discography"), lang.L("Top Tracks")})
	viewToggle.SetActivatedLabel(a.activeView)
	viewToggle.OnChanged = a.onViewChange
	a.sortButton = widgets.NewSortChooserButton(util.LocalizeSlice(discographySorts), func(selIdx int) {
		a.cfg.DiscographySort = discographySorts[selIdx]
		a.showAlbumGrid(true /*reSort*/)
	})
	a.sortButton.AlignLeft = true
	for i, sort := range discographySorts {
		if sort == a.cfg.DiscographySort {
			a.sortButton.SetSelectedIndex(i)
		}
	}
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

var _ CanSelectAll = (*ArtistPage)(nil)

func (a *ArtistPage) SelectAll() {
	if a.activeView == 1 && a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).SelectAll()
	}
}

func (a *ArtistPage) UnselectAll() {
	if a.tracklistCtr != nil {
		a.tracklistCtr.Objects[0].(*widgets.Tracklist).UnselectAll()
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
		s.listScrollPos = tl.GetScrollOffset()
		s.trackSort = tl.Sorting()
		tl.Clear()
		a.pool.Release(util.WidgetTypeTracklist, tl)
	}
	a.header.artistPage = nil
	a.pool.Release(util.WidgetTypeArtistPageHeader, a.header)
	if a.albumGrid != nil {
		s.gridScrollPos = a.albumGrid.GetScrollOffset()
		a.albumGrid.Clear()
		a.pool.Release(util.WidgetTypeGridView, a.albumGrid)
	}
	if a.groupedReleases != nil {
		s.gridScrollPos = a.groupedReleases.GetScrollOffset()
		s.sectionVis = a.groupedReleases.GetSectionVisibility()
		s.sectionVisNeedApply = true
		a.groupedReleases.Model = widgets.GroupedReleasesModel{}
		a.pool.Release(util.WidgetTypeGroupedReleases, a.groupedReleases)
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
	} else if g.activeView == 0 && g.groupedReleases != nil {
		g.groupedReleases.ScrollToOffset(g.groupedReleases.GetScrollOffset() + scrollAmt)
	} else if g.activeView == 1 && g.tracklistCtr != nil {
		tl := g.tracklistCtr.Objects[0].(*widgets.Tracklist)
		tl.ScrollBy(scrollAmt)
	}
}

func (a *ArtistPage) playArtistRadio() {
	go func() {
		err := a.pm.PlaySimilarSongs(a.artistID)
		if err != nil {
			fyne.Do(func() {
				log.Printf("error playing similar songs: %v", err)
				a.contr.ToastProvider.ShowErrorToast(lang.L("Unable to play artist radio"))
			})
		}
	}()
}

func (a *ArtistPage) getGridViewAlbumsModel() []widgets.GridViewItemModel {
	if a.artistInfo == nil {
		return nil
	}

	a.sortAlbumsSlices(a.artistInfo.Albums)
	return sharedutil.MapSlice(a.artistInfo.Albums, a.albumToGridViewItemModel)
}

func (a *ArtistPage) albumToGridViewItemModel(al *mediaprovider.Album) widgets.GridViewItemModel {
	return widgets.GridViewItemModel{
		Name:        al.Name,
		ID:          al.ID,
		CoverArtID:  al.CoverArtID,
		Secondary:   []string{strconv.Itoa(al.YearOrZero())},
		CanFavorite: true,
		IsFavorite:  al.Favorite,
	}
}

func (a *ArtistPage) getGroupedReleasesModel() widgets.GroupedReleasesModel {
	if a.artistInfo == nil {
		return widgets.GroupedReleasesModel{}
	}
	albums := []*mediaprovider.Album{}
	compilations := []*mediaprovider.Album{}
	eps := []*mediaprovider.Album{}
	singles := []*mediaprovider.Album{}

	for _, album := range a.artistInfo.Albums {
		switch rt := album.ReleaseTypes; {
		case rt&mediaprovider.ReleaseTypeEP > 0:
			eps = append(eps, album)
		case rt&mediaprovider.ReleaseTypeCompilation > 0:
			compilations = append(compilations, album)
		case rt&mediaprovider.ReleaseTypeSingle > 0:
			singles = append(singles, album)
		default:
			albums = append(albums, album)
		}
	}

	a.sortAlbumsSlices(albums, compilations, eps, singles)
	return widgets.GroupedReleasesModel{
		Albums:       sharedutil.MapSlice(albums, a.albumToGridViewItemModel),
		Compilations: sharedutil.MapSlice(compilations, a.albumToGridViewItemModel),
		EPs:          sharedutil.MapSlice(eps, a.albumToGridViewItemModel),
		Singles:      sharedutil.MapSlice(singles, a.albumToGridViewItemModel),
	}
}

func (a *ArtistPage) sortAlbumsSlices(slices ...[]*mediaprovider.Album) {
	var curSlice []*mediaprovider.Album
	sortFunc := func(x, y int) bool {
		return curSlice[y].Date.After(curSlice[x].Date)
	}
	switch a.cfg.DiscographySort {
	case discographySorts[1]: /*year descending*/
		sortFunc = func(x, y int) bool {
			return curSlice[x].Date.After(curSlice[y].Date)
		}
	case discographySorts[2]: /*name*/
		sortFunc = func(x, y int) bool {
			return curSlice[x].Name < curSlice[y].Name
		}
	}

	for _, slice := range slices {
		curSlice = slice
		sort.Slice(slice, sortFunc)
	}
}

// should be called asynchronously
func (a *ArtistPage) load() {
	artist, err := a.mp.GetArtist(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist: %s", err.Error())
		return
	}

	fyne.Do(func() {
		if a.disposed {
			return
		}
		a.artistInfo = artist
		a.header.Update(artist, a.im)
		if a.activeView == 0 {
			a.showAlbumGrid(false /*reSort*/)
		} else {
			go a.showTopTracks()
		}
	})

	info, err := a.mp.GetArtistInfo(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist info: %s", err.Error())
	}
	fyne.Do(func() {
		if !a.disposed {
			a.header.UpdateInfo(info)
		}
	})
}

func (a *ArtistPage) showAlbumGrid(reSort bool) {
	allAlbums := func() bool {
		return slices.IndexFunc(a.artistInfo.Albums, func(al *mediaprovider.Album) bool {
			return al.ReleaseTypes&mediaprovider.ReleaseTypeCompilation > 0 ||
				al.ReleaseTypes&mediaprovider.ReleaseTypeEP > 0 ||
				al.ReleaseTypes&mediaprovider.ReleaseTypeSingle > 0
		}) < 0
	}
	useGroupedReleases := a.artistInfo != nil && len(a.artistInfo.Albums) <= 50 && !allAlbums()

	if a.albumGrid == nil && a.groupedReleases == nil {
		if a.artistInfo == nil {
			// page not loaded yet or invalid artist
			a.activeView = 0 // if page still loading, will show discography view first
			return
		}
		if useGroupedReleases {
			model := a.getGroupedReleasesModel()
			if g := a.pool.Obtain(util.WidgetTypeGroupedReleases); g != nil {
				a.groupedReleases = g.(*widgets.GroupedReleases)
				a.groupedReleases.Model = model
			} else {
				a.groupedReleases = widgets.NewGroupedReleases(model, a.im)
			}
			a.contr.ConnectGroupedReleasesActions(a.groupedReleases)
			if a.sectionVisNeedApply {
				a.groupedReleases.SetSectionVisibility(a.sectionVis, true)
				a.sectionVisNeedApply = false
			}
			if a.gridScrollPos != 0 {
				a.groupedReleases.ScrollToOffset(a.gridScrollPos)
				a.gridScrollPos = 0
			}
		} else {
			model := a.getGridViewAlbumsModel()
			if g := a.pool.Obtain(util.WidgetTypeGridView); g != nil {
				a.albumGrid = g.(*widgets.GridView)
				a.albumGrid.Placeholder = myTheme.AlbumIcon
				a.albumGrid.ResetFixed(model)
			} else {
				a.albumGrid = widgets.NewFixedGridView(model, a.im, myTheme.AlbumIcon)
			}
			a.contr.ConnectAlbumGridActions(a.albumGrid)
			if a.gridScrollPos != 0 {
				a.albumGrid.ScrollToOffset(a.gridScrollPos)
				a.gridScrollPos = 0
			}
		}
	} else if reSort {
		if useGroupedReleases {
			a.groupedReleases.Model = a.getGroupedReleasesModel()
		} else {
			model := a.getGridViewAlbumsModel()
			a.albumGrid.ResetFixed(model)
		}
	}
	a.sortButton.Show()
	if useGroupedReleases {
		a.container.Objects[0].(*fyne.Container).Objects[0] = a.groupedReleases
		a.container.Objects[0].Refresh()
	} else {
		a.container.Objects[0].(*fyne.Container).Objects[0] = a.albumGrid
	}
	a.container.Objects[0].Refresh()
}

// should be called asynchronously
func (a *ArtistPage) showTopTracks() {
	updated := false
	updatePage := func() {
		a.sortButton.Hide()
		a.container.Objects[0].(*fyne.Container).Objects[0] = a.tracklistCtr
		a.container.Objects[0].Refresh()
	}

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
		updated = true // mark that updatePage() will be called here
		fyne.Do(func() {
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
			if a.listScrollPos != 0 {
				tl.ScrollToOffset(a.listScrollPos)
				a.listScrollPos = 0
			}
			a.tracklistCtr = container.New(
				&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, BottomPadding: 10},
				tl)
			updatePage()
		})
	}

	if !updated {
		fyne.Do(updatePage)
	}
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
	return newArtistPage(*s)
}

const artistBioNotAvailableKey = "Artist biography not available."

type ArtistPageHeader struct {
	widget.BaseWidget

	artistID              string
	artistPage            *ArtistPage
	artistImage           *widgets.ImagePlaceholder
	artistImageID         string
	titleDisp             *widget.RichText
	biographyDisp         *widgets.MaxRowsLabel
	similarArtists        *fyne.Container
	favoriteBtn           *widgets.FavoriteButton
	playBtn               *widget.Button
	playRadioBtn          *widget.Button
	menuBtn               *widget.Button
	container             *fyne.Container
	fullSizeCoverFetching bool
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
	a.artistImage.OnTapped = func(*fyne.PointEvent) { a.showPopUpCover() }
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
	a.artistImageID = ""
	a.favoriteBtn.IsFavorited = false
	a.titleDisp.Segments[0].(*widget.TextSegment).Text = ""
	a.biographyDisp.Text = lang.L(artistBioNotAvailableKey)
	for _, obj := range a.similarArtists.Objects {
		obj.Hide()
	}
	a.artistImage.SetImage(nil, false)
	a.fullSizeCoverFetching = false
}

func (a *ArtistPageHeader) Update(artist *mediaprovider.ArtistWithAlbums, im *backend.ImageManager) {
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
	a.artistImageID = artist.CoverArtID
	go func() {
		if cover, err := im.GetCoverThumbnail(artist.CoverArtID); err == nil {
			fyne.Do(func() { a.artistImage.SetImage(cover, true) })
		} else {
			log.Printf("error fetching cover: %v", err)
		}
	}()
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
			go func() {
				im, err := a.artistPage.im.FetchAndCacheArtistImage(a.artistID, info.ImageURL)
				if err == nil {
					fyne.Do(func() { a.artistImage.SetImage(im, true /*tappable*/) })
				}
			}()
		}
	}
}

// should NOT be called asynchronously
func (a *ArtistPageHeader) showPopUpCover() {
	if a.artistImageID == "" {
		if im := a.artistImage.Image(); im != nil {
			a.artistPage.contr.ShowPopUpImage(im)
		}
	} else {
		if a.fullSizeCoverFetching {
			return
		}
		a.fullSizeCoverFetching = true
		go func() {
			defer func() { a.fullSizeCoverFetching = false }()
			cover, err := a.artistPage.im.GetFullSizeCoverArt(a.artistImageID)
			if err != nil {
				log.Printf("error getting full size album cover: %s", err.Error())
				return
			}
			if a.artistPage != nil {
				fyne.Do(func() { a.artistPage.contr.ShowPopUpImage(cover) })
			}
		}()
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
