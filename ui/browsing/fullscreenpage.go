package browsing

import (
	"context"
	"image"
	"log"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	"github.com/dweymouth/supersonic/ui/layouts"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type FullscreenPage struct {
	widget.BaseWidget

	fullscreenPageState

	queue           []*mediaprovider.Track
	imageLoadCancel context.CancelFunc
	card            *widgets.LargeNowPlayingCard
	nowPlayingID    string
	albumID         string
}

type fullscreenPageState struct {
	contr   *controller.Controller
	pool    *util.WidgetPool
	pm      *backend.PlaybackManager
	im      *backend.ImageManager
	canRate bool
}

func NewFullscreenPage(
	contr *controller.Controller,
	pool *util.WidgetPool,
	im *backend.ImageManager,
	pm *backend.PlaybackManager,
	canRate bool,
) *FullscreenPage {
	a := &FullscreenPage{fullscreenPageState: fullscreenPageState{
		contr: contr, pool: pool, im: im, pm: pm, canRate: canRate,
	}}
	a.ExtendBaseWidget(a)

	a.card = widgets.NewLargeNowPlayingCard()
	a.card.OnAlbumNameTapped = func() { contr.NavigateTo(controller.AlbumRoute(a.albumID)) }
	a.card.OnArtistNameTapped = func(artistID string) { contr.NavigateTo(controller.ArtistRoute(artistID)) }

	a.Reload()
	return a
}

func (a *FullscreenPage) CreateRenderer() fyne.WidgetRenderer {
	paddedLayout := &layouts.PercentPadLayout{
		LeftRightObjectPercent: .8,
		TopBottomObjectPercent: .8,
	}
	container := container.NewGridWithColumns(2,
		container.New(paddedLayout, a.card),
		container.New(paddedLayout,
			util.AddHeaderBackground(
				container.NewAppTabs(
					container.NewTabItem("Up Next", layout.NewSpacer()),
					container.NewTabItem("Lyrics", layout.NewSpacer()),
				))),
	)
	return widget.NewSimpleRenderer(container)
}

func (a *FullscreenPage) Save() SavedPage {
	if a.imageLoadCancel != nil {
		a.imageLoadCancel()
	}
	nps := a.fullscreenPageState
	return &nps
}

func (a *FullscreenPage) Route() controller.Route {
	return controller.NowPlayingRoute("")
}

var _ CanShowNowPlaying = (*FullscreenPage)(nil)

func (a *FullscreenPage) OnSongChange(song, lastScrobbledIfAny *mediaprovider.Track) {
	if a.imageLoadCancel != nil {
		a.imageLoadCancel()
	}
	a.albumID = song.AlbumID
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(song)
	a.card.Update(song.Name, song.ArtistNames, song.ArtistIDs, song.Album)
	a.imageLoadCancel = a.im.GetFullSizeCoverArtAsync(song.CoverArtID, func(img image.Image, err error) {
		if err != nil {
			log.Printf("error loading cover art: %v\n", err)
		} else {
			a.card.SetCoverImage(img)
		}
	})
}

func (a *FullscreenPage) Reload() {
	a.queue = a.pm.GetPlayQueue()
}

func (s *fullscreenPageState) Restore() Page {
	return NewFullscreenPage(s.contr, s.pool, s.im, s.pm, s.canRate)
}
