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
	sm       *backend.ServerManager
	im       *backend.ImageManager
	nav      func(Route)
	contr    controller.Controller
}

type ArtistPage struct {
	widget.BaseWidget

	artistPageState

	header    *ArtistPageHeader
	container *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewArtistPage(artistID string, sm *backend.ServerManager, im *backend.ImageManager, contr controller.Controller, nav func(Route)) *ArtistPage {
	a := &ArtistPage{artistPageState: artistPageState{
		artistID: artistID,
		sm:       sm,
		im:       im,
		nav:      nav,
		contr:    contr,
	}}
	a.ExtendBaseWidget(a)
	a.header = NewArtistPageHeader(a, nav)
	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, layout.NewSpacer())
	go a.load()
	return a
}

func (a *ArtistPage) Route() Route {
	return ArtistRoute(a.artistID)
}

func (a *ArtistPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *ArtistPage) Reload() {
	go a.load()
}

func (a *ArtistPage) Save() SavedPage {
	s := a.artistPageState
	return &s
}

func (a *ArtistPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID, 0)
	}
}

func (a *ArtistPage) onShowAlbumPage(albumID string) {
	a.nav(AlbumRoute(albumID))
}

// should be called asynchronously
func (a *ArtistPage) load() {
	artist, err := a.sm.Server.GetArtist(a.artistID)
	if err != nil {
		log.Printf("Failed to get artist: %s", err.Error())
		return
	}
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
	return NewArtistPage(s.artistID, s.sm, s.im, s.contr, s.nav)
}

type ArtistPageHeader struct {
	widget.BaseWidget

	nav            func(Route)
	artistID       string
	artistPage     *ArtistPage
	artistImage    *widgets.ImagePlaceholder
	titleDisp      *widget.RichText
	biographyDisp  *widget.RichText
	similarArtists *fyne.Container
	container      *fyne.Container
}

func NewArtistPageHeader(page *ArtistPage, nav func(Route)) *ArtistPageHeader {
	a := &ArtistPageHeader{
		nav:            nav,
		artistPage:     page,
		titleDisp:      widget.NewRichTextWithText(""),
		biographyDisp:  widget.NewRichTextWithText("Artist biography not available."),
		similarArtists: container.NewHBox(),
	}
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistImage = widgets.NewImagePlaceholder(res.ResPeopleInvertPng, 225)
	a.biographyDisp.Wrapping = fyne.TextWrapWord
	a.ExtendBaseWidget(a)
	a.createContainer()
	return a
}

func (a *ArtistPageHeader) Update(artist *subsonic.ArtistID3) {
	if artist == nil {
		return
	}
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
			return func() { a.nav(ArtistRoute(id)) }
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

func (a *ArtistPageHeader) createContainer() {
	a.container = container.NewBorder(nil, nil, a.artistImage, nil,
		container.NewBorder(a.titleDisp, nil, nil, nil, container.NewVBox(a.biographyDisp, a.similarArtists)))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}
