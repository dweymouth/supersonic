package browsing

import (
	"log"
	"strconv"
	"strings"

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

	cfg   *backend.ArtistPageConfig
	pm    *backend.PlaybackManager
	mp    mediaprovider.MediaProvider
	im    *backend.ImageManager
	contr *controller.Controller
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState

	artistInfo *mediaprovider.ArtistWithAlbums

	albumGrid    *widgets.GridView
	tracklistCtr *fyne.Container
	nowPlayingID string
	header       *ArtistPageHeader
	container    *fyne.Container
}

func NewArtistPage(artistID string, cfg *backend.ArtistPageConfig, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager, contr *controller.Controller) *ArtistPage {
	activeView := 0
	if cfg.InitialView == "Top Tracks" {
		activeView = 1
	}
	return newArtistPage(artistID, cfg, pm, mp, im, contr, activeView, widgets.TracklistSort{})
}

func newArtistPage(artistID string, cfg *backend.ArtistPageConfig, pm *backend.PlaybackManager, mp mediaprovider.MediaProvider, im *backend.ImageManager, contr *controller.Controller, activeView int, sort widgets.TracklistSort) *ArtistPage {
	a := &ArtistPage{artistPageState: artistPageState{
		artistID:   artistID,
		cfg:        cfg,
		pm:         pm,
		mp:         mp,
		im:         im,
		contr:      contr,
		activeView: activeView,
		trackSort:  sort,
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
	s := a.artistPageState
	if a.tracklistCtr != nil {
		s.trackSort = a.tracklistCtr.Objects[0].(*widgets.Tracklist).Sorting()
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
				Secondary:  strconv.Itoa(al.Year),
			}
		})
		a.albumGrid = widgets.NewFixedGridView(model, a.im, myTheme.AlbumIcon)
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
		tl := widgets.NewTracklist(ts)
		tl.AutoNumber = true
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
	return newArtistPage(s.artistID, s.cfg, s.pm, s.mp, s.im, s.contr, s.activeView, s.trackSort)
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
	a.playRadioBtn = widget.NewButtonWithIcon(" Play Artist Radio", myTheme.ShuffleIcon, page.playArtistRadio)
	a.biographyDisp.Wrapping = fyne.TextWrapWord
	a.ExtendBaseWidget(a)
	a.createContainer()
	return a
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
}

func (a *ArtistPageHeader) UpdateInfo(info *mediaprovider.ArtistInfo) {
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
	for i, art := range info.SimilarArtists {
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
	a.container = container.NewBorder(nil, nil, a.artistImage, nil,
		container.NewVBox(
			container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
				a.titleDisp, a.biographyDisp, a.similarArtists),
			container.NewHBox(util.NewHSpace(2), a.favoriteBtn, a.playBtn, a.playRadioBtn)))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
