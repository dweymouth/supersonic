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

	disposed     bool
	header       *PlaylistPageHeader
	tracklist    *widgets.Tracklist
	tracks       []*mediaprovider.Track
	nowPlayingID string
	container    *fyne.Container
}

type playlistPageState struct {
	playlistID string
	conf       *backend.PlaylistPageConfig
	contr      *controller.Controller
	widgetPool *util.WidgetPool
	sm         *backend.ServerManager
	pm         *backend.PlaybackManager
	im         *backend.ImageManager
	trackSort  widgets.TracklistSort
}

func NewPlaylistPage(
	playlistID string,
	conf *backend.PlaylistPageConfig,
	pool *util.WidgetPool,
	contr *controller.Controller,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	im *backend.ImageManager,
) *PlaylistPage {
	return newPlaylistPage(playlistID, conf, contr, pool, sm, pm, im, widgets.TracklistSort{})
}

func newPlaylistPage(
	playlistID string,
	conf *backend.PlaylistPageConfig,
	contr *controller.Controller,
	pool *util.WidgetPool,
	sm *backend.ServerManager,
	pm *backend.PlaybackManager,
	im *backend.ImageManager,
	trackSort widgets.TracklistSort,
) *PlaylistPage {
	a := &PlaylistPage{playlistPageState: playlistPageState{playlistID: playlistID, conf: conf, contr: contr, widgetPool: pool, sm: sm, pm: pm, im: im}}
	a.ExtendBaseWidget(a)
	if h := a.widgetPool.Obtain(util.WidgetTypePlaylistPageHeader); h != nil {
		a.header = h.(*PlaylistPageHeader)
		a.header.page = a
		a.header.Clear()
	} else {
		a.header = NewPlaylistPageHeader(a)
	}
	a.header.page = a
	if tl := a.widgetPool.Obtain(util.WidgetTypeTracklist); tl != nil {
		a.tracklist = tl.(*widgets.Tracklist)
		a.tracklist.Reset()
	} else {
		a.tracklist = widgets.NewTracklist(nil)
	}
	a.tracklist.SetVisibleColumns(conf.TracklistColumns)
	a.tracklist.SetSorting(trackSort)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		conf.TracklistColumns = cols
	}
	reorderMenu := fyne.NewMenuItem("Reorder tracks", nil)
	reorderMenu.ChildMenu = fyne.NewMenu("", []*fyne.MenuItem{
		fyne.NewMenuItem("Move to top", a.onMoveSelectedToTop),
		fyne.NewMenuItem("Move up", a.onMoveSelectedUp),
		fyne.NewMenuItem("Move down", a.onMoveSelectedDown),
		fyne.NewMenuItem("Move to bottom", a.onMoveSelectedToBottom),
	}...)
	a.tracklist.Options = widgets.TracklistOptions{
		AuxiliaryMenuItems: []*fyne.MenuItem{reorderMenu,
			fyne.NewMenuItem("Remove from playlist", a.onRemoveSelectedFromPlaylist)},
	}
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
	a.disposed = true
	p := a.playlistPageState
	p.trackSort = a.tracklist.Sorting()
	p.widgetPool.Release(util.WidgetTypePlaylistPageHeader, a.header)
	a.tracklist.Clear()
	p.widgetPool.Release(util.WidgetTypeTracklist, a.tracklist)
	return &p
}

func (a *PlaylistPage) Route() controller.Route {
	return controller.PlaylistRoute(a.playlistID)
}

func (a *PlaylistPage) OnSongChange(track, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlayingID = sharedutil.TrackIDOrEmptyStr(track)
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
	if a.disposed {
		return
	}
	renumberTracks(playlist.Tracks)
	a.tracks = playlist.Tracks
	a.tracklist.SetTracks(playlist.Tracks)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.header.Update(playlist)
}

func renumberTracks(tracks []*mediaprovider.Track) {
	// Playlists, like albums, have a sequential running order. We want the number column to
	// represent the track's original position in the playlist, if the user applies a sort.
	// Re-purpose the TrackNumber field for the track's position in the playlist, rather than its parent album.
	for i, tr := range tracks {
		tr.TrackNumber = i + 1
	}
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
	// Since the tracklist view may be sorted in a different order than the
	// actual running order, we need to get the IDs of the selected tracks
	// from the tracklist and convert them to indices in the *original* run order
	ids := a.tracklist.SelectedTrackIDs()
	idxs := make([]int, 0, len(ids))
	for i, tr := range a.tracks {
		if sharedutil.SliceContains(ids, tr.ID) {
			idxs = append(idxs, i)
		}
	}
	newTracks := sharedutil.ReorderTracks(a.tracks, idxs, op)
	ids = sharedutil.TracksToIDs(newTracks)
	if err := a.sm.Server.ReplacePlaylistTracks(a.playlistID, ids); err != nil {
		log.Printf("error updating playlist: %s", err.Error())
	} else {
		renumberTracks(newTracks)
		// force-switch back to unsorted view to show new track order
		a.tracklist.SetSorting(widgets.TracklistSort{})
		a.tracklist.SetTracks(newTracks)
		a.tracklist.UnselectAll()
	}
}

func (a *PlaylistPage) onRemoveSelectedFromPlaylist() {
	sel := sharedutil.ToSet(a.tracklist.SelectedTrackIDs())
	idxs := make([]int, 0, len(sel))
	for i, tr := range a.tracks {
		if _, ok := sel[tr.ID]; ok {
			idxs = append(idxs, i)
		}
	}
	a.sm.Server.EditPlaylistTracks(a.playlistID, nil, idxs)
	a.tracklist.UnselectAll()
	a.Reload()
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
	// due to widget reuse a.page can change so page MUST NOT
	// be directly captured in a closure throughout this function!
	a := &PlaylistPageHeader{page: page}
	a.ExtendBaseWidget(a)

	a.image = widgets.NewImagePlaceholder(myTheme.PlaylistIcon, 225)
	a.titleLabel = widget.NewRichTextWithText("")
	a.titleLabel.Truncation = fyne.TextTruncateEllipsis
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.descriptionLabel = widget.NewLabel("")
	a.ownerLabel = widget.NewLabel("")
	a.createdAtLabel = widget.NewLabel("")
	a.trackTimeLabel = widget.NewLabel("")
	a.editButton = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		if a.playlistInfo != nil {
			a.page.contr.DoEditPlaylistWorkflow(&a.playlistInfo.Playlist)
		}
	})
	a.editButton.Hidden = true
	playButton := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		a.page.pm.LoadTracks(a.page.tracks, false, false)
		a.page.pm.PlayFromBeginning()
	})
	// TODO: find way to pad shuffle svg rather than using a space in the label string
	shuffleBtn := widget.NewButtonWithIcon(" Shuffle", myTheme.ShuffleIcon, func() {
		a.page.pm.LoadTracks(a.page.tracks, false /*append*/, true /*shuffle*/)
		a.page.pm.PlayFromBeginning()
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
						sharedutil.TracksToIDs(a.page.tracks))
				}),
				fyne.NewMenuItem("Download...", func() {
					a.page.contr.ShowDownloadDialog(a.page.tracks, a.playlistInfo.Name)
				}))
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+menuBtn.Size().Height))
	}

	a.container = util.AddHeaderBackground(
		container.NewBorder(nil, nil, a.image, nil,
			container.NewVBox(a.titleLabel, container.New(&layouts.VboxCustomPadding{ExtraPad: -10},
				a.descriptionLabel,
				a.ownerLabel,
				a.trackTimeLabel),
				container.NewHBox(a.editButton, playButton, shuffleBtn, menuBtn),
			)))
	return a
}

func (a *PlaylistPageHeader) Clear() {
	a.titleLabel.Segments[0].(*widget.TextSegment).Text = ""
	a.createdAtLabel.Text = ""
	a.descriptionLabel.Text = ""
	a.ownerLabel.Text = ""
	a.image.SetImage(nil, false)
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
	return newPlaylistPage(s.playlistID, s.conf, s.contr, s.widgetPool, s.sm, s.pm, s.im, s.trackSort)
}
