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
	queueList       *widgets.PlayQueueList
	imageLoadCancel context.CancelFunc
	card            *widgets.LargeNowPlayingCard
	nowPlayingID    string
	albumID         string
	container       *fyne.Container
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
	a.card.DisableRating = !canRate
	a.card.OnAlbumNameTapped = func() {
		contr.NavigateTo(controller.AlbumRoute(a.albumID))
	}
	a.card.OnArtistNameTapped = func(artistID string) {
		contr.NavigateTo(controller.ArtistRoute(artistID))
	}
	a.card.OnSetFavorite = func(fav bool) {
		a.contr.SetTrackFavorites([]string{a.nowPlayingID}, fav)
	}
	a.card.OnSetRating = func(rating int) {
		a.contr.SetTrackRatings([]string{a.nowPlayingID}, rating)
	}

	a.queueList = widgets.NewPlayQueueList(a.im)
	paddedLayout := &layouts.PercentPadLayout{
		LeftRightObjectPercent: .8,
		TopBottomObjectPercent: .8,
	}
	a.container = container.NewGridWithColumns(2,
		container.New(paddedLayout, a.card),
		container.New(paddedLayout,
			util.AddHeaderBackground(
				container.NewAppTabs(
					container.NewTabItem("Play Queue",
						container.NewBorder(layout.NewSpacer(), nil, nil, nil, a.queueList)),
					container.NewTabItem("Lyrics", layout.NewSpacer()),
				))),
	)

	a.Reload()
	return a
}

func (a *FullscreenPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
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
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(song)
	a.queueList.SetNowPlaying(a.nowPlayingID)

	a.albumID = sharedutil.AlbumIDOrEmptyStr(song)
	a.card.Update(song)
	if song == nil {
		a.card.SetCoverImage(nil)
		return
	}
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
	a.queueList.SetTracks(a.queue)
}

func (s *fullscreenPageState) Restore() Page {
	return NewFullscreenPage(s.contr, s.pool, s.im, s.pm, s.canRate)
}
