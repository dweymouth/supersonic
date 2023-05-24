package browsing

import (
	"fmt"
	"log"

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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	conf       *backend.PlaylistPageConfig
	contr      *controller.Controller
	sm         *backend.ServerManager
	pm         *backend.PlaybackManager
	im         *backend.ImageManager
}

func NewPlaylistPage(
	playlistID string,
	conf *backend.PlaylistPageConfig,
	contr *controller.Controller,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	im *backend.ImageManager,
) *PlaylistPage {
	a := &PlaylistPage{playlistPageState: playlistPageState{playlistID: playlistID, conf: conf, contr: contr, sm: sm, pm: pm, im: im}}
	a.ExtendBaseWidget(a)
	a.header = NewPlaylistPageHeader(a)
	a.tracklist = widgets.NewTracklist(nil)
	a.tracklist.SetVisibleColumns(conf.TracklistColumns)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		conf.TracklistColumns = cols
	}
	a.tracklist.AutoNumber = true
	reorderMenu := fyne.NewMenuItem("Reorder tracks", nil)
	reorderMenu.ChildMenu = fyne.NewMenu("", []*fyne.MenuItem{
		fyne.NewMenuItem("Move to top", a.onMoveSelectedToTop),
		fyne.NewMenuItem("Move up", a.onMoveSelectedUp),
		fyne.NewMenuItem("Move down", a.onMoveSelectedDown),
		fyne.NewMenuItem("Move to bottom", a.onMoveSelectedToBottom),
	}...)
	a.tracklist.AuxiliaryMenuItems = []*fyne.MenuItem{reorderMenu,
		fyne.NewMenuItem("Remove from playlist", a.onRemoveSelectedFromPlaylist)}
	// connect tracklist actions
	a.contr.ConnectTracklistActions(a.tracklist)

	a.container = container.NewBorder(
		container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadTop: 15, PadBottom: 10}, a.header),
		nil, nil, nil, container.New(&layouts.MaxPadLayout{PadLeft: 15, PadRight: 15, PadBottom: 15}, a.tracklist))
	go a.load()
	return a
}

func (a *PlaylistPage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *PlaylistPage) Save() SavedPage {
	p := a.playlistPageState
	return &p
}

func (a *PlaylistPage) Route() controller.Route {
	return controller.PlaylistRoute(a.playlistID)
}

func (a *PlaylistPage) OnSongChange(song, lastScrobbledIfAny *mediaprovider.Track) {
	if song == nil {
		a.nowPlayingID = ""
	} else {
		a.nowPlayingID = song.ID
	}
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.TrackIDOrEmptyStr(lastScrobbledIfAny))
}

func (a *PlaylistPage) Reload() {
	go a.load()
}

func (a *PlaylistPage) Tapped(*fyne.PointEvent) {
	a.tracklist.UnselectAll()
}

func (a *PlaylistPage) SelectAll() {
	a.tracklist.SelectAll()
}

// should be called asynchronously
func (a *PlaylistPage) load() {
	playlist, err := a.sm.Server.GetPlaylist(a.playlistID)
	if err != nil {
		log.Printf("Failed to get playlist: %s", err.Error())
		return
	}
	a.tracklist.Tracks = playlist.Tracks
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.Refresh()
	a.header.Update(playlist)
}

func (a *PlaylistPage) onMoveSelectedToTop() {
	a.doSetNewTrackOrder(sharedutil.MoveToTop)
}

func (a *PlaylistPage) onMoveSelectedUp() {
	a.doSetNewTrackOrder(sharedutil.MoveUp)
}

func (a *PlaylistPage) onMoveSelectedDown() {
	a.doSetNewTrackOrder(sharedutil.MoveDown)
}

func (a *PlaylistPage) onMoveSelectedToBottom() {
	a.doSetNewTrackOrder(sharedutil.MoveToBottom)
}

func (a *PlaylistPage) doSetNewTrackOrder(op sharedutil.TrackReorderOp) {
	idxs := a.tracklist.SelectedTrackIndexes()
	newTracks := sharedutil.ReorderTracks(a.tracklist.Tracks, idxs, op)
	ids := make([]string, len(newTracks))
	for i, tr := range newTracks {
		ids[i] = tr.ID
	}
	if err := a.sm.Server.ReplacePlaylistTracks(a.playlistID, ids); err != nil {
		log.Printf("error updating playlist: %s", err.Error())
	} else {
		a.tracklist.Tracks = newTracks
		a.tracklist.UnselectAll()
		a.tracklist.Refresh()
	}
}

func (a *PlaylistPage) onRemoveSelectedFromPlaylist() {
	a.sm.Server.EditPlaylistTracks(a.playlistID, nil, a.tracklist.SelectedTrackIndexes())
	a.tracklist.UnselectAll()
	go a.Reload()
}

type PlaylistPageHeader struct {
	widget.BaseWidget

	page         *PlaylistPage
	playlistInfo *mediaprovider.PlaylistWithTracks
	image        *widgets.ImagePlaceholder

	editButton       *widget.Button
	titleLabel       *widget.RichText
	descriptionLabel *widget.Label
	createdAtLabel   *widget.Label
	ownerLabel       *widget.Label
	trackTimeLabel   *widget.Label

	container *fyne.Container
}

func NewPlaylistPageHeader(page *PlaylistPage) *PlaylistPageHeader {
	a := &PlaylistPageHeader{page: page}
	a.ExtendBaseWidget(a)

	a.image = widgets.NewImagePlaceholder(myTheme.PlaylistIcon, 225)
	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Wrapping = fyne.TextTruncate
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.descriptionLabel = widget.NewLabel("")
	a.ownerLabel = widget.NewLabel("")
	a.createdAtLabel = widget.NewLabel("")
	a.trackTimeLabel = widget.NewLabel("")
	a.editButton = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		if a.playlistInfo != nil {
			page.contr.DoEditPlaylistWorkflow(&a.playlistInfo.Playlist)
		}
	})
	a.editButton.Hidden = true
	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		page.pm.LoadTracks(page.tracklist.Tracks, false, false)
		page.pm.PlayFromBeginning()
	})
	// TODO: find way to pad shuffle svg rather than using a space in the label string
	shuffleBtn := widget.NewButtonWithIcon(" Shuffle", myTheme.ShuffleIcon, func() {
		page.pm.LoadTracks(page.tracklist.Tracks, false /*append*/, true /*shuffle*/)
		page.pm.PlayFromBeginning()
	})
	var pop *widget.PopUpMenu
	menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuBtn.OnTapped = func() {
		if pop == nil {
			menu := fyne.NewMenu("",
				fyne.NewMenuItem("Add to queue", func() {
					a.page.pm.LoadPlaylist(a.page.playlistID, true /*append*/, false /*shuffle*/)
				}),
				fyne.NewMenuItem("Add to playlist...", func() {
					a.page.contr.DoAddTracksToPlaylistWorkflow(
						sharedutil.TracksToIDs(a.page.tracklist.Tracks))
				}))
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+menuBtn.Size().Height))
	}

	a.container = container.NewBorder(nil, nil, a.image, nil,
		container.NewVBox(a.titleLabel, container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
			a.descriptionLabel,
			a.ownerLabel,
			a.trackTimeLabel),
			container.NewHBox(a.editButton, playButton, shuffleBtn, menuBtn),
		))
	return a
}

func (a *PlaylistPageHeader) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.container)
}

func (a *PlaylistPageHeader) Update(playlist *mediaprovider.PlaylistWithTracks) {
	a.playlistInfo = playlist
	a.editButton.Hidden = playlist.Owner != a.page.sm.LoggedInUser
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = playlist.Name
	a.descriptionLabel.SetText(playlist.Description)
	a.ownerLabel.SetText(a.formatPlaylistOwnerStr(playlist))
	a.trackTimeLabel.SetText(a.formatPlaylistTrackTimeStr(playlist))
	a.createdAtLabel.SetText("created at TODO")

	var haveCover bool
	if playlist.CoverArtID != "" {
		if im, err := a.page.im.GetCoverThumbnail(playlist.CoverArtID); err == nil && im != nil {
			a.image.SetImage(im, false /*tappable*/)
			haveCover = true
		}
	}
	if !haveCover {
		if im, err := a.page.im.GetCoverThumbnail(playlist.ID); err == nil && im != nil {
			a.image.SetImage(im, false)
		}
	}
	a.Refresh()
}

func (a *PlaylistPageHeader) formatPlaylistOwnerStr(p *mediaprovider.PlaylistWithTracks) string {
	pubPriv := "Public"
	if !p.Public {
		pubPriv = "Private"
	}
	return fmt.Sprintf("%s playlist by %s", pubPriv, p.Owner)
}

func (a *PlaylistPageHeader) formatPlaylistTrackTimeStr(p *mediaprovider.PlaylistWithTracks) string {
	tracks := "tracks"
	if p.TrackCount == 1 {
		tracks = "track"
	}
	return fmt.Sprintf("%d %s, %s", p.TrackCount, tracks, util.SecondsToTimeString(float64(p.Duration)))
}

func (s *playlistPageState) Restore() Page {
	return NewPlaylistPage(s.playlistID, s.conf, s.contr, s.sm, s.pm, s.im)
}
