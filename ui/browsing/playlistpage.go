package browsing

import (
	"fmt"
	"log"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/controller"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
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
		a.tracklist = widgets.NewTracklist(nil, a.im, false)
	}
	a.tracklist.SetVisibleColumns(conf.TracklistColumns)
	a.tracklist.SetSorting(trackSort)
	a.tracklist.OnVisibleColumnsChanged = func(cols []string) {
		conf.TracklistColumns = cols
	}
	a.tracklist.OnReorderTracks = a.doSetNewTrackOrder
	_, canRate := a.sm.Server.(mediaprovider.SupportsRating)
	_, canShare := a.sm.Server.(mediaprovider.SupportsSharing)
	remove := fyne.NewMenuItem(lang.L("Remove from playlist"), a.onRemoveSelectedFromPlaylist)
	remove.Icon = theme.ContentClearIcon()
	a.tracklist.Options = widgets.TracklistOptions{
		Reorderable:        true,
		DisableRating:      !canRate,
		DisableSharing:     !canShare,
		AuxiliaryMenuItems: []*fyne.MenuItem{remove},
	}
	// connect tracklist actions
	a.contr.ConnectTracklistActions(a.tracklist)

	a.container = container.NewBorder(
		container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, TopPadding: 15, BottomPadding: 10}, a.header),
		nil, nil, nil, container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15, BottomPadding: 15}, a.tracklist))
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
	a.tracklist.OnReorderTracks = nil
	p.widgetPool.Release(util.WidgetTypeTracklist, a.tracklist)
	return &p
}

func (a *PlaylistPage) Route() controller.Route {
	return controller.PlaylistRoute(a.playlistID)
}

var _ CanShowNowPlaying = (*PlaylistPage)(nil)

func (a *PlaylistPage) OnSongChange(item mediaprovider.MediaItem, lastScrobbledIfAny *mediaprovider.Track) {
	a.nowPlayingID = sharedutil.MediaItemIDOrEmptyStr(item)
	a.tracklist.SetNowPlaying(a.nowPlayingID)
	a.tracklist.IncrementPlayCount(sharedutil.MediaItemIDOrEmptyStr(lastScrobbledIfAny))
}

func (a *PlaylistPage) Reload() {
	go a.load()
}

var _ CanSelectAll = (*PlaylistPage)(nil)

func (a *PlaylistPage) SelectAll() {
	a.tracklist.SelectAll()
}

func (a *PlaylistPage) UnselectAll() {
	a.tracklist.UnselectAll()
}

var _ Scrollable = (*PlaylistPage)(nil)

func (a *PlaylistPage) Scroll(scrollAmt float32) {
	a.tracklist.Scroll(scrollAmt)
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

func (a *PlaylistPage) doSetNewTrackOrder(ids []string, newPos int) {
	// Since the tracklist view may be sorted in a different order than the
	// actual running order, we need to get the IDs of the selected tracks
	// from the tracklist and convert them to indices in the *original* run order
	idSet := sharedutil.ToSet(ids)
	idxs := make([]int, 0, len(idSet))
	for i, tr := range a.tracks {
		if _, ok := idSet[tr.ID]; ok {
			idxs = append(idxs, i)
		}
	}
	newTracks := sharedutil.ReorderItems(a.tracks, idxs, newPos)
	// we can't block the UI waiting for the server so assume it will succeed
	go func() {
		ids = sharedutil.TracksToIDs(newTracks)
		if err := a.sm.Server.ReplacePlaylistTracks(a.playlistID, ids); err != nil {
			log.Printf("error updating playlist: %s", err.Error())
		}
	}()

	renumberTracks(newTracks)
	// force-switch back to unsorted view to show new track order
	a.tracklist.SetSorting(widgets.TracklistSort{})
	a.tracklist.SetTracks(newTracks)
	a.tracklist.UnselectAll()
	a.tracks = newTracks
}

func (a *PlaylistPage) onRemoveSelectedFromPlaylist() {
	a.sm.Server.RemovePlaylistTracks(a.playlistID, a.tracklist.SelectedTrackIndexes())
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

	fullSizeCoverFetching bool

	container *fyne.Container
}

func NewPlaylistPageHeader(page *PlaylistPage) *PlaylistPageHeader {
	// due to widget reuse a.page can change so page MUST NOT
	// be directly captured in a closure throughout this function!
	a := &PlaylistPageHeader{page: page}
	a.ExtendBaseWidget(a)

	a.image = widgets.NewImagePlaceholder(myTheme.PlaylistIcon, 225)
	a.image.OnTapped = func(*fyne.PointEvent) { go a.showPopUpCover() }
	a.titleLabel = util.NewTruncatingRichText()
	a.titleLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyle{
		SizeName: theme.SizeNameHeadingText,
	}
	a.descriptionLabel = util.NewTruncatingLabel()
	a.ownerLabel = util.NewTruncatingLabel()
	a.createdAtLabel = widget.NewLabel("")
	a.trackTimeLabel = widget.NewLabel("")
	a.editButton = widget.NewButtonWithIcon(lang.L("Edit"), theme.DocumentCreateIcon(), func() {
		if a.playlistInfo != nil {
			a.page.contr.DoEditPlaylistWorkflow(&a.playlistInfo.Playlist)
		}
	})
	a.editButton.Hidden = true
	playButton := widget.NewButtonWithIcon(lang.L("Play"), theme.MediaPlayIcon(), func() {
		a.page.pm.LoadTracks(a.page.tracks, backend.Replace, false)
		a.page.pm.PlayFromBeginning()
	})
	shuffleBtn := widget.NewButtonWithIcon(lang.L("Shuffle"), myTheme.ShuffleIcon, func() {
		a.page.pm.LoadTracks(a.page.tracks, backend.Replace, true)
		a.page.pm.PlayFromBeginning()
	})
	var pop *widget.PopUpMenu
	menuBtn := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuBtn.OnTapped = func() {
		if pop == nil {
			playNext := fyne.NewMenuItem(lang.L("Play next"), func() {
				go a.page.pm.LoadPlaylist(a.page.playlistID, backend.InsertNext, false)
			})
			playNext.Icon = myTheme.PlayNextIcon
			queue := fyne.NewMenuItem(lang.L("Add to queue"), func() {
				go a.page.pm.LoadPlaylist(a.page.playlistID, backend.Append, false)
			})
			queue.Icon = theme.ContentAddIcon()
			playlist := fyne.NewMenuItem(lang.L("Add to playlist")+"...", func() {
				a.page.contr.DoAddTracksToPlaylistWorkflow(
					sharedutil.TracksToIDs(a.page.tracks))
			})
			playlist.Icon = myTheme.PlaylistIcon
			download := fyne.NewMenuItem(lang.L("Download")+"...", func() {
				a.page.contr.ShowDownloadDialog(a.page.tracks, a.titleLabel.String())
			})
			download.Icon = theme.DownloadIcon()
			menu := fyne.NewMenu("", playNext, queue, playlist, download)
			pop = widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(a))
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuBtn)
		pop.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+menuBtn.Size().Height))
	}

	a.container = util.AddHeaderBackground(
		container.NewBorder(nil, nil, a.image, nil,
			container.NewVBox(a.titleLabel, container.New(layout.NewCustomPaddedVBoxLayout(theme.Padding()-10),
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
			a.image.SetImage(im, true /*tappable*/)
			haveCover = true
		}
	}
	if !haveCover {
		if im, err := a.page.im.GetCoverThumbnail(playlist.ID); err == nil && im != nil {
			a.image.SetImage(im, true)
		}
	}
	a.Refresh()
}

func (a *PlaylistPageHeader) showPopUpCover() {
	if a.fullSizeCoverFetching || a.playlistInfo == nil {
		return
	}
	a.fullSizeCoverFetching = true
	defer func() { a.fullSizeCoverFetching = false }()
	cover, err := a.page.im.GetFullSizeCoverArt(a.playlistInfo.CoverArtID)
	if err != nil {
		log.Printf("error getting full size playlist cover: %s", err.Error())
		return
	}
	if a.page != nil {
		a.page.contr.ShowPopUpImage(cover)
	}
}

func (a *PlaylistPageHeader) formatPlaylistOwnerStr(p *mediaprovider.PlaylistWithTracks) string {
	pubPriv := lang.L("Public playlist by")
	if !p.Public {
		pubPriv = lang.L("Private playlist by")
	}
	return fmt.Sprintf("%s %s", pubPriv, p.Owner)
}

func (a *PlaylistPageHeader) formatPlaylistTrackTimeStr(p *mediaprovider.PlaylistWithTracks) string {
	var tracks string
	if p.TrackCount == 1 {
		tracks = lang.L("track")
	} else {
		tracks = lang.L("tracks")
	}
	return fmt.Sprintf("%d %s, %s", p.TrackCount, tracks, util.SecondsToTimeString(float64(p.Duration)))
}

func (s *playlistPageState) Restore() Page {
	return newPlaylistPage(s.playlistID, s.conf, s.contr, s.widgetPool, s.sm, s.pm, s.im, s.trackSort)
}
