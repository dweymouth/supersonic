package browsing

import (
	"log"
	"strings"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/sharedutil"
	"supersonic/ui/controller"
	"supersonic/ui/layouts"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type artistPageState struct {
	artistID   string
	activeView int

	cfg   *backend.ArtistPageConfig
	pm    *backend.PlaybackManager
	sm    *backend.ServerManager
	im    *backend.ImageManager
	contr *controller.Controller
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState

	artistInfo *subsonic.ArtistID3

	albumGrid    *widgets.AlbumGrid
	tracklistCtr *fyne.Container
	nowPlayingID string
	header       *ArtistPageHeader
	container    *fyne.Container
}

func NewArtistPage(artistID string, cfg *backend.ArtistPageConfig, pm *backend.PlaybackManager, sm *backend.ServerManager, im *backend.ImageManager, contr *controller.Controller) *ArtistPage {
	activeView := 0
	if cfg.InitialView == "Top Tracks" {
		activeView = 1
	}
	return newArtistPage(artistID, cfg, pm, sm, im, contr, activeView)
}

func newArtistPage(artistID string, cfg *backend.ArtistPageConfig, pm *backend.PlaybackManager, sm *backend.ServerManager, im *backend.ImageManager, contr *controller.Controller, activeView int) *ArtistPage {
	a := &ArtistPage{artistPageState: artistPageState{
		artistID:   artistID,
		cfg:        cfg,
		pm:         pm,
		sm:         sm,
		im:         im,
		contr:      contr,
		activeView: activeView,
	}}
	a.ExtendBaseWidget(a)
	a.header = NewArtistPageHeader(a)
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
	// TODO: find a better place to update the tracklist columns preference
	// If user changes columns but doesn't navigate to another page,
	// we won't be persisting the change
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		a.cfg.TracklistColumns = tl.VisibleColumns()
	}
	s := a.artistPageState
	return &s
}

var _ CanShowNowPlaying = (*ArtistPage)(nil)

func (a *ArtistPage) OnSongChange(track *subsonic.Child, lastScrobbledIfAny *subsonic.Child) {
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(track)
	if a.tracklistCtr != nil {
		tl := a.tracklistCtr.Objects[0].(*widgets.Tracklist)
		tl.SetNowPlaying(a.nowPlayingID)
		tl.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
	}
}

func (a *ArtistPage) onPlayAlbum(albumID string) {
	a.pm.PlayAlbum(albumID, 0)
}

func (a *ArtistPage) playAllTracks() {
	if a.artistInfo != nil { // page loaded
		for i, album := range a.artistInfo.Album {
			a.pm.LoadAlbum(album.ID, i > 0 /*append*/, false /*shuffle*/)
		}
		a.pm.PlayFromBeginning()
	}
}

func (a *ArtistPage) playArtistRadio() {
	go a.pm.PlaySimilarSongs(a.artistID)
}

func (a *ArtistPage) onShowAlbumPage(albumID string) {
	a.contr.NavigateTo(controller.AlbumRoute(albumID))
}

// should be called asynchronously
func (a *ArtistPage) load() {
	artist, err := a.sm.Server.GetArtist(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist: %s", err.Error())
		return
	}
	a.artistInfo = artist
	a.header.Update(artist)
	if a.activeView == 0 {
		a.showAlbumGrid()
	} else {
		a.showTopTracks()
	}
	info, err := a.sm.Server.GetArtistInfo2(a.artistID, nil)
	if err != nil {
		log.Printf("Failed to get artist info: %s", err.Error())
	}
	a.header.UpdateInfo(info)
}

func (a *ArtistPage) showAlbumGrid() {
	if a.albumGrid == nil {
		a.albumGrid = widgets.NewFixedAlbumGrid(a.artistInfo.Album, a.im, true /*showYear*/)
		a.albumGrid.OnPlayAlbum = a.onPlayAlbum
		a.albumGrid.OnShowAlbumPage = a.onShowAlbumPage
	}
	a.container.Objects[0].(*fyne.Container).Objects[0] = a.albumGrid
	a.container.Objects[0].Refresh()
}

func (a *ArtistPage) showTopTracks() {
	if a.tracklistCtr == nil {
		ts, err := a.sm.Server.GetTopSongs(a.artistInfo.Name, map[string]string{"count": "20"})
		if err != nil {
			log.Printf("error getting top songs: %s", err.Error())
			return
		}
		tl := widgets.NewTracklist(ts)
		tl.AutoNumber = true
		tl.SetVisibleColumns(a.cfg.TracklistColumns)
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
	return newArtistPage(s.artistID, s.cfg, s.pm, s.sm, s.im, s.contr, s.activeView)
}

type ArtistPageHeader struct {
	widget.BaseWidget

	artistID       string
	artistPage     *ArtistPage
	artistImage    *widgets.ImagePlaceholder
	titleDisp      *widget.RichText
	biographyDisp  *widget.RichText
	similarArtists *fyne.Container
	favoriteBtn    *widgets.FavoriteButton
	playBtn        *widget.Button
	playRadioBtn   *widget.Button
	container      *fyne.Container
}

func NewArtistPageHeader(page *ArtistPage) *ArtistPageHeader {
	a := &ArtistPageHeader{
		artistPage:     page,
		titleDisp:      widget.NewRichTextWithText(""),
		biographyDisp:  widget.NewRichTextWithText("Artist biography not available."),
		similarArtists: container.NewHBox(),
	}
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistImage = widgets.NewImagePlaceholder(res.ResPeopleInvertPng, 225)
	a.favoriteBtn = widgets.NewFavoriteButton(func() { go a.toggleFavorited() })
	a.playBtn = widget.NewButtonWithIcon("Play Discography", theme.MediaPlayIcon(), page.playAllTracks)
	a.playRadioBtn = widget.NewButtonWithIcon("Play Artist Radio", res.ResShuffleInvertSvg, page.playArtistRadio)
	a.biographyDisp.Wrapping = fyne.TextWrapWord
	a.ExtendBaseWidget(a)
	a.createContainer()
	return a
}

func (a *ArtistPageHeader) Update(artist *subsonic.ArtistID3) {
	if artist == nil {
		return
	}
	a.favoriteBtn.IsFavorited = !artist.Starred.IsZero()
	a.favoriteBtn.Refresh()
	a.artistID = artist.ID
	a.titleDisp.Segments[0].(*widget.TextSegment).Text = artist.Name
	a.titleDisp.Refresh()
}

func (a *ArtistPageHeader) UpdateInfo(info *subsonic.ArtistInfo2) {
	if info == nil {
		return
	}

	if info.Biography != "" {
		segs := util.RichTextSegsFromHTMLString(info.Biography)
		if len(segs) > 0 {
			if ts, ok := segs[0].(*widget.TextSegment); ok && strings.TrimSpace(ts.Text) != "" {
				a.biographyDisp.Segments = segs
				a.biographyDisp.Refresh()
			}
		}
	}

	a.similarArtists.RemoveAll()
	for i, art := range info.SimilarArtist {
		if i == 0 {
			a.similarArtists.Add(widget.NewLabel("Similar Artists:"))
		}
		if i == 4 {
			break
		}
		h := widgets.NewCustomHyperlink()
		h.NoTruncate = true
		h.SetText(art.Name)
		h.OnTapped = func(id string) func() {
			return func() { a.artistPage.contr.NavigateTo(controller.ArtistRoute(id)) }
		}(art.ID)
		a.similarArtists.Add(h)
	}
	a.similarArtists.Refresh()

	if info.LargeImageUrl != "" {
		if a.artistImage.HaveImage() {
			_ = a.artistPage.im.RefreshCachedArtistImageIfExpired(a.artistID, info.LargeImageUrl)
		} else {
			im, err := a.artistPage.im.FetchAndCacheArtistImage(a.artistID, info.LargeImageUrl)
			if err == nil {
				a.artistImage.SetImage(im, true /*tappable*/)
			}
		}
	}
}

func (a *ArtistPageHeader) toggleFavorited() {
	if a.favoriteBtn.IsFavorited {
		a.artistPage.sm.Server.Star(subsonic.StarParameters{ArtistIDs: []string{a.artistID}})
	} else {
		a.artistPage.sm.Server.Unstar(subsonic.StarParameters{ArtistIDs: []string{a.artistID}})
	}
}

func (a *ArtistPageHeader) createContainer() {
	a.container = container.NewBorder(nil, nil, a.artistImage, nil,
		container.NewVBox(
			container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
				a.titleDisp, a.biographyDisp, a.similarArtists),
			container.NewHBox(util.NewHSpace(2), a.favoriteBtn, a.playBtn, a.playRadioBtn)))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
