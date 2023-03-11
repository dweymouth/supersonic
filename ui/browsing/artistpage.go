package browsing

import (
	"bytes"
	"image"
	"log"
	"strings"
	"supersonic/backend"
	"supersonic/res"
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
	artistID string
	pm       *backend.PlaybackManager
	sm       *backend.ServerManager
	im       *backend.ImageManager
	contr    *controller.Controller
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState

	artistInfo *subsonic.ArtistID3

	header    *ArtistPageHeader
	container *fyne.Container
}

func NewArtistPage(artistID string, pm *backend.PlaybackManager, sm *backend.ServerManager, im *backend.ImageManager, contr *controller.Controller) *ArtistPage {
	a := &ArtistPage{artistPageState: artistPageState{
		artistID: artistID,
		pm:       pm,
		sm:       sm,
		im:       im,
		contr:    contr,
	}}
	a.ExtendBaseWidget(a)
	a.header = NewArtistPageHeader(a)
	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, layout.NewSpacer())
	go a.load()
	return a
}

func (a *ArtistPage) Route() controller.Route {
	return controller.ArtistRoute(a.artistID)
}

func (a *ArtistPage) Reload() {
	go a.load()
}

func (a *ArtistPage) Save() SavedPage {
	s := a.artistPageState
	return &s
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
	ag := widgets.NewFixedAlbumGrid(artist.Album, a.im, true /*showYear*/)
	ag.OnPlayAlbum = a.onPlayAlbum
	ag.OnShowAlbumPage = a.onShowAlbumPage
	a.container.Objects[0] = ag
	a.container.Refresh()
	info, err := a.sm.Server.GetArtistInfo2(a.artistID, nil)
	if err != nil {
		log.Printf("Failed to get artist info: %s", err.Error())
	}
	a.header.UpdateInfo(info)
}

func (a *ArtistPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

func (s *artistPageState) Restore() Page {
	return NewArtistPage(s.artistID, s.pm, s.sm, s.im, s.contr)
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
		if res, err := fyne.LoadResourceFromURLString(info.LargeImageUrl); err == nil {
			im, _, err := image.Decode(bytes.NewReader(res.Content()))
			if err != nil {
				return
			}
			a.artistImage.OnTapped = func() {
				a.artistPage.contr.ShowPopUpImage(im)
			}
			a.artistImage.SetImage(im, true /*tappable*/)
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
			container.NewHBox(widgets.NewHSpace(2), a.favoriteBtn, a.playBtn, a.playRadioBtn)))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
