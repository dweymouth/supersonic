package browsing

import (
	"fmt"
	"log"
	"supersonic/backend"
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

	playlistID    string
	sm            *backend.ServerManager
	pm            *backend.PlaybackManager
	nav           func(Route)
	header        *PlaylistPageHeader
	tracklist     *widgets.Tracklist
	nowPlayingID  string
	container     *fyne.Container
	popUpProvider PopUpProvider
}

func NewPlaylistPage(
	playlistID string,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	nav func(Route),
) *PlaylistPage {
	a := &PlaylistPage{playlistID: playlistID, sm: sm, pm: pm}
	a.ExtendBaseWidget(a)
	a.header = NewPlaylistPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.AutoNumber = true
	a.tracklist.OnPlayTrackAt = a.onPlayTrackAt
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
	return &savedPlaylistPage{
		playlistID: a.playlistID,
		sm:         a.sm,
		pm:         a.pm,
		nav:        a.nav,
	}
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
		a.header.Update(playlist)
		a.tracklist.Tracks = playlist.Entry
		a.tracklist.SetNowPlaying(a.nowPlayingID)
	}()
}

type PlaylistPageHeader struct {
	widget.BaseWidget

	page *PlaylistPage

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

	a.container = container.NewVBox(a.titleLabel, container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
		a.descriptionLabel,
		a.ownerLabel,
		a.trackTimeLabel),
		container.NewHBox(a.playButton),
	)
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

type savedPlaylistPage struct {
	playlistID string
	sm         *backend.ServerManager
	pm         *backend.PlaybackManager
	nav        func(Route)
}

func (s *savedPlaylistPage) Restore() Page {
	return NewPlaylistPage(s.playlistID, s.sm, s.pm, s.nav)
}
