package dialogs

import (
	"fmt"
	"image"
	"net/url"
	"strings"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/ui/layouts"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const musicBrainzReleaseUrl = "https://musicbrainz.org/release"

type AlbumInfoDialog struct {
	widget.BaseWidget

	OnDismiss func()

	content fyne.CanvasObject
}

func NewAlbumInfoDialog(albumInfo *mediaprovider.AlbumInfo, albumName string, albumCover image.Image) *AlbumInfoDialog {
	a := &AlbumInfoDialog{}
	a.ExtendBaseWidget(a)

	a.content = container.NewVBox(
		a.buildMainContainer(albumInfo, albumName, albumCover),
		widget.NewSeparator(),
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewButton("Close", func() {
				if a.OnDismiss != nil {
					a.OnDismiss()
				}
			}),
		),
	)
	return a
}

func (a *AlbumInfoDialog) MinSize() fyne.Size {
	return fyne.NewSize(550, a.BaseWidget.MinSize().Height)
}

func (a *AlbumInfoDialog) buildMainContainer(albumInfo *mediaprovider.AlbumInfo, albumName string, albumCover image.Image) *fyne.Container {
	iconImage := canvas.NewImageFromImage(albumCover)
	iconImage.FillMode = canvas.ImageFillContain
	iconImage.SetMinSize(fyne.NewSize(100, 100))
	iconImage.Hidden = albumCover == nil
	title := widget.NewRichTextWithText(albumName)
	title.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = true
	title.Segments[0].(*widget.TextSegment).Style.SizeName = theme.SizeNameSubHeadingText
	title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter

	infoContent := widget.NewLabel("Album info not available")

	if albumInfo.Notes != "" {
		infoContent = a.infoLabel(albumInfo.Notes)
	}

	urlContainer := a.buildUrlContainer(albumInfo.LastFmUrl, albumInfo.MusicBrainzID)

	return container.New(
		&layouts.MaxPadLayout{PadLeft: 15, PadRight: 10, PadTop: 15, PadBottom: 10},
		container.NewVBox(
			iconImage,
			title,
			infoContent,
			urlContainer,
		),
	)

}

func (a *AlbumInfoDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.content)
}

func (a *AlbumInfoDialog) infoLabel(info string) *widget.Label {
	last := strings.LastIndex(info, "<a href")
	infoWithoutLink := strings.TrimSpace(info[:last])
	lbl := widget.NewLabel(infoWithoutLink)
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Alignment = fyne.TextAlignLeading
	return lbl
}

func (a *AlbumInfoDialog) buildUrlContainer(lastFm, musicBrainzID string) *fyne.Container {
	urls := make([]*widget.Hyperlink, 0)

	if lastFm != "" {
		if lastFmUrl, err := url.Parse(lastFm); err == nil {
			urls = append(urls, widget.NewHyperlink("Last.fm", lastFmUrl))
		}
	}

	if musicBrainzID != "" {
		if musicBrainzUrl, err := url.Parse(fmt.Sprintf("%s/%s", musicBrainzReleaseUrl, musicBrainzID)); err == nil {
			urls = append(urls, widget.NewHyperlink("MusicBrainz", musicBrainzUrl))
		}
	}

	urlContainer := container.New(&layouts.HboxCustomPadding{DisableThemePad: true, ExtraPad: -10})
	for index, url := range urls {
		if index > 0 {
			urlContainer.Add(widget.NewLabel("·"))
		}
		urlContainer.Add(url)
	}

	return container.NewCenter(urlContainer)
}
