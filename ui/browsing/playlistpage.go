package browsing

import (
	"fmt"
	"log"
	"supersonic/backend"
	"supersonic/res"
	"supersonic/ui/layouts"
	"supersonic/ui/util"
	"supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type PlaylistPage struct {
	widget.BaseWidget

	playlistPageState

	header       *PlaylistPageHeader
	tracklist    *widgets.Tracklist
	nowPlayingID string
	container    *fyne.Container
}

type playlistPageState struct {
	playlistID string
	sm         *backend.ServerManager
	pm         *backend.PlaybackManager
	im         *backend.ImageManager
	nav        func(Route)
}

func NewPlaylistPage(
	playlistID string,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	im *backend.ImageManager,
	nav func(Route),
) *PlaylistPage {
	a := &PlaylistPage{playlistPageState: playlistPageState{playlistID: playlistID, sm: sm, pm: pm, im: im}}
	a.ExtendBaseWidget(a)
	a.header = NewPlaylistPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.AutoNumber = true
	// connect tracklist actions
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
	a.tracklist.OnAddToQueue = func(tracks []*subsonic.Child) { a.pm.LoadTracks(tracks, true) }
	a.tracklist.OnPlaySelection = func(tracks []*subsonic.Child) {
		a.pm.LoadTracks(tracks, false)
		a.pm.PlayFromBeginning()
	}

	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadBottom: 15}, a.tracklist))
	a.loadAsync()
	return a
}

func (a *PlaylistPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *PlaylistPage) Save() SavedPage {
	p := a.playlistPageState
	return &p
}

func (a *PlaylistPage) Route() Route {
	return AlbumRoute(a.playlistID)
}

func (a *PlaylistPage) OnSongChange(song *subsonic.Child) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
}

func (a *PlaylistPage) Reload() {
	a.loadAsync()
}

func (a *PlaylistPage) Tapped(*fyne.PointEvent) {
	a.tracklist.UnselectAll()
}

func (a *PlaylistPage) onPlayTrackAt(tracknum int) {
	a.pm.PlayPlaylist(a.playlistID, tracknum)
}

func (a *PlaylistPage) loadAsync() {
	go func() {
		playlist, err := a.sm.Server.GetPlaylist(a.playlistID)
		if err != nil {
			log.Printf("Failed to get playlist: %s", err.Error())
			return
		}
		a.tracklist.Tracks = playlist.Entry
		a.tracklist.SetNowPlaying(a.nowPlayingID)
		a.tracklist.Refresh()
		a.header.Update(playlist)
	}()
}

type PlaylistPageHeader struct {
	widget.BaseWidget

	page *PlaylistPage

	image *widgets.ImagePlaceholder

	titleLabel       *widget.RichText
	descriptionLabel *widget.Label
	createdAtLabel   *widget.Label
	ownerLabel       *widget.Label
	trackTimeLabel   *widget.Label

	playButton *widget.Button

	container *fyne.Container
}

func NewPlaylistPageHeader(page *PlaylistPage) *PlaylistPageHeader {
	a := &PlaylistPageHeader{page: page}
	a.ExtendBaseWidget(a)

	a.image = widgets.NewImagePlaceholder(res.ResPlaylistInvertPng, 225)
	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Wrapping = fyne.TextTruncate
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.descriptionLabel = widget.NewLabel("")
	a.ownerLabel = widget.NewLabel("")
	a.createdAtLabel = widget.NewLabel("")
	a.trackTimeLabel = widget.NewLabel("")
	a.playButton = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		page.onPlayTrackAt(0)
	})

	a.container = container.NewBorder(nil, nil, a.image, nil,
		container.NewVBox(a.titleLabel, container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
			a.descriptionLabel,
			a.ownerLabel,
			a.trackTimeLabel),
			container.NewHBox(a.playButton),
		))
	return a
}

func (a *PlaylistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *PlaylistPageHeader) Update(playlist *subsonic.Playlist) {
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = playlist.Name
	a.descriptionLabel.SetText(playlist.Comment)
	a.ownerLabel.SetText(a.formatPlaylistOwnerStr(playlist))
	a.trackTimeLabel.SetText(a.formatPlaylistTrackTimeStr(playlist))
	a.createdAtLabel.SetText("created at TODO")

	var haveCover bool
	if playlist.CoverArt != "" {
		if im, err := a.page.im.GetAlbumThumbnail(playlist.CoverArt); err == nil && im != nil {
			a.image.SetImage(im, false /*tappable*/)
			haveCover = true
		}
	}
	if !haveCover {
		if im, err := a.page.im.GetAlbumThumbnail(playlist.ID); err == nil && im != nil {
			a.image.SetImage(im, false)
		}
	}
	a.Refresh()
}

func (a *PlaylistPageHeader) formatPlaylistOwnerStr(p *subsonic.Playlist) string {
	pubPriv := "Public"
	if !p.Public {
		pubPriv = "Private"
	}
	return fmt.Sprintf("%s playlist by %s", pubPriv, p.Owner)
}

func (a *PlaylistPageHeader) formatPlaylistTrackTimeStr(p *subsonic.Playlist) string {
	return fmt.Sprintf("%d tracks, %s", p.SongCount, util.SecondsToTimeString(float64(p.Duration)))
}

func (s *playlistPageState) Restore() Page {
	return NewPlaylistPage(s.playlistID, s.sm, s.pm, s.im, s.nav)
}
