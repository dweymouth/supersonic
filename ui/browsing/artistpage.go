package browsing

import (
	"bytes"
	"image"
	"image/color"
	"log"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/layouts"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

var _ fyne.Widget = (*ArtistPage)(nil)

type ArtistPage struct {
	widget.BaseWidget

	artistID  string
	im        *backend.ImageManager
	sm        *backend.ServerManager
	nav       func(Route)
	grid      *widgets.AlbumGrid
	header    *ArtistPageHeader
	container *fyne.Container

	OnPlayAlbum func(string, int)
}

func NewArtistPage(artistID string, sm *backend.ServerManager, im *backend.ImageManager, nav func(Route)) *ArtistPage {
	a := &ArtistPage{
		artistID: artistID,
		sm:       sm,
		im:       im,
		nav:      nav,
	}
	a.ExtendBaseWidget(a)
	a.header = NewArtistPageHeader()
	a.container = container.NewBorder(a.header, nil, nil, nil, layout.NewSpacer())
	a.loadAsync()
	return a
}

func (a *ArtistPage) Route() Route {
	return ArtistRoute(a.artistID)
}

func (a *ArtistPage) SetPlayAlbumCallback(cb func(string, int)) {
	a.OnPlayAlbum = cb
}

func (a *ArtistPage) Reload() {
	a.loadAsync()
}

func (a *ArtistPage) Save() SavedPage {
	return &savedArtistPage{
		artistID: a.artistID,
		sm:       a.sm,
		im:       a.im,
		nav:      a.nav,
	}
}

func (a *ArtistPage) onPlayAlbum(albumID string) {
	if a.OnPlayAlbum != nil {
		a.OnPlayAlbum(albumID, 0)
	}
}

func (a *ArtistPage) onShowAlbumPage(albumID string) {
	a.nav(AlbumRoute(albumID))
}

func (a *ArtistPage) loadAsync() {
	go func() {
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
	}()
}

func (a *ArtistPage) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)
	return widget.NewSimpleRenderer(a.container)
}

type savedArtistPage struct {
	artistID string
	sm       *backend.ServerManager
	im       *backend.ImageManager
	nav      func(Route)
}

func (s *savedArtistPage) Restore() Page {
	return NewArtistPage(s.artistID, s.sm, s.im, s.nav)
}

type ArtistPageHeader struct {
	widget.BaseWidget

	artistID       string
	artistImageCtr *fyne.Container
	titleDisp      *widget.RichText
	biographyDisp  *widget.Label
	similarArtists *widget.RichText
	container      *fyne.Container
}

func NewArtistPageHeader() *ArtistPageHeader {
	a := &ArtistPageHeader{
		titleDisp:      widget.NewRichTextWithText(""),
		biographyDisp:  widget.NewLabel("Artist description not available"),
		similarArtists: widget.NewRichText(),
	}
	a.titleDisp.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.artistImageCtr = container.New(&layouts.CenterPadLayout{PadLeftRight: 10, PadTopBottom: 10},
		NewMissingArtistImage())
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
		a.biographyDisp.SetText(info.Biography)
	}
	/** TODO:
	if len(info.SimilarArtist) > 0 {
		segments := make([]widget.RichTextSegment, 0)
		segments = append(segments, &widget.TextSegment{Text: "Similar artists: "})
		for sim := info.SimilarArtist {
			segments = append(segments, &widget.HyperlinkSegment{

			})
		}
	}
	*/
	if info.LargeImageUrl != "" {
		if res, err := fyne.LoadResourceFromURLString(info.LargeImageUrl); err == nil {
			im, _, err := image.Decode(bytes.NewReader(res.Content()))
			if err != nil {
				return
			}
			img := canvas.NewImageFromImage(im)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(225, 225))
			a.artistImageCtr.RemoveAll()
			a.artistImageCtr.Add(img)
			a.artistImageCtr.Refresh()
		}
	}
}

func (a *ArtistPageHeader) createContainer() {
	a.container = container.NewBorder(nil, nil, a.artistImageCtr, nil,
		container.NewBorder(a.titleDisp, a.similarArtists, nil, nil, a.biographyDisp))
}

func (a *ArtistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

type MissingArtistImage struct {
	widget.BaseWidget
	container *fyne.Container
}

func NewMissingArtistImage() *MissingArtistImage {
	m := &MissingArtistImage{}
	m.ExtendBaseWidget(m)
	img := canvas.NewImageFromResource(res.ResPeopleInvertPng)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(64, 64))
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = color.Black
	rect.StrokeWidth = 3
	rect.SetMinSize(fyne.NewSize(225, 225))
	m.container = container.NewMax(
		container.NewCenter(img),
		rect,
	)
	return m
}

func (m *MissingArtistImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.container)
}
