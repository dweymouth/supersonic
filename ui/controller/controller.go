package controller

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/player"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/dialogs"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type NavigationHandler func(Route)

type ReloadFunc func()

type CurPageFunc func() Route

type Controller struct {
	AppVersion  string
	MainWindow  fyne.Window
	App         *backend.App
	NavHandler  NavigationHandler
	CurPageFunc CurPageFunc
	ReloadFunc  ReloadFunc

	escapablePopUp   *widget.PopUp
	haveModal        bool
	runOnModalClosed func()
}

func (m *Controller) NavigateTo(route Route) {
	m.NavHandler(route)
}

func (m *Controller) ClosePopUpOnEscape(pop *widget.PopUp) {
	m.escapablePopUp = pop
}

func (m *Controller) CloseEscapablePopUp() {
	if m.escapablePopUp != nil {
		m.escapablePopUp.Hide()
		m.escapablePopUp = nil
		m.doModalClosed()
	}
}

// If there is currently no modal popup managed by the Controller visible,
// then run f (which should create and show a modal dialog) immediately.
// else run f when the current modal dialog workflow has ended.
func (m *Controller) QueueShowModalFunc(f func()) {
	if m.haveModal {
		m.runOnModalClosed = f
	} else {
		f()
	}
}

func (m *Controller) ShowPopUpImage(img image.Image) {
	im := canvas.NewImageFromImage(img)
	im.FillMode = canvas.ImageFillContain
	pop := widget.NewPopUp(im, m.MainWindow.Canvas())
	s := m.MainWindow.Canvas().Size()
	var popS fyne.Size
	if asp := util.ImageAspect(img); s.Width/s.Height > asp {
		// window height is limiting factor
		h := s.Height * 0.8
		popS = fyne.NewSize(h*asp, h)
	} else {
		w := s.Width * 0.8
		popS = fyne.NewSize(w, w*(1/asp))
	}
	m.ClosePopUpOnEscape(pop)
	pop.Resize(popS)
	pop.ShowAtPosition(fyne.NewPos(
		(s.Width-popS.Width)/2,
		(s.Height-popS.Height)/2,
	))
}

func (m *Controller) ConnectTracklistActions(tracklist *widgets.Tracklist) {
	tracklist.OnAddToPlaylist = m.DoAddTracksToPlaylistWorkflow
	tracklist.OnAddToQueue = func(tracks []*mediaprovider.Track) {
		m.App.PlaybackManager.LoadTracks(tracks, true, false)
	}
	tracklist.OnPlayTrackAt = func(idx int) {
		m.App.PlaybackManager.LoadTracks(tracklist.Tracks, false, false)
		m.App.PlaybackManager.PlayTrackAt(idx)
	}
	tracklist.OnPlaySelection = func(tracks []*mediaprovider.Track, shuffle bool) {
		m.App.PlaybackManager.LoadTracks(tracks, false, shuffle)
		m.App.PlaybackManager.PlayFromBeginning()
	}
	tracklist.OnSetFavorite = m.SetTrackFavorites
	tracklist.OnSetRating = m.SetTrackRatings
	tracklist.OnShowAlbumPage = func(albumID string) {
		m.NavigateTo(AlbumRoute(albumID))
	}
	tracklist.OnShowArtistPage = func(artistID string) {
		m.NavigateTo(ArtistRoute(artistID))
	}
	tracklist.OnColumnVisibilityMenuShown = func(pop *widget.PopUp) {
		m.ClosePopUpOnEscape(pop)
	}
}

func (m *Controller) ConnectAlbumGridActions(grid *widgets.GridView) {
	grid.OnAddToQueue = func(albumID string) {
		m.App.PlaybackManager.LoadAlbum(albumID, true, false)
	}
	grid.OnPlay = func(albumID string, shuffle bool) {
		m.App.PlaybackManager.PlayAlbum(albumID, 0, shuffle)
	}
	grid.OnShowItemPage = func(albumID string) {
		m.NavigateTo(AlbumRoute(albumID))
	}
	grid.OnShowSecondaryPage = func(artistID string) {
		m.NavigateTo(ArtistRoute(artistID))
	}
	grid.OnAddToPlaylist = func(albumID string) {
		go func() {
			album, err := m.App.ServerManager.Server.GetAlbum(albumID)
			if err != nil {
				log.Printf("error loading album: %s", err.Error())
				return
			}
			m.DoAddTracksToPlaylistWorkflow(sharedutil.TracksToIDs(album.Tracks))
		}()
	}
}

func (m *Controller) PromptForFirstServer() {
	d := dialogs.NewAddEditServerDialog("Connect to Server", false, nil, m.MainWindow.Canvas().Focus)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func() {
		d.DisableSubmit()
		go func() {
			if m.testConnectionAndUpdateDialogText(d) {
				// connection is good
				pop.Hide()
				m.doModalClosed()
				conn := backend.ServerConnection{
					Hostname:    d.Host,
					AltHostname: d.AltHost,
					Username:    d.Username,
					LegacyAuth:  d.LegacyAuth,
				}
				server := m.App.Config.AddServer(d.Nickname, conn)
				if err := m.trySetPasswordAndConnectToServer(server, d.Password); err != nil {
					log.Printf("error connecting to server: %s", err.Error())
				}
			}
			d.EnableSubmit()
		}()
	}
	m.haveModal = true
	pop.Show()
}

// Show dialog to prompt for playlist.
// Depending on the results of that dialog, potentially create a new playlist
// Add tracks to the user-specified playlist
func (m *Controller) DoAddTracksToPlaylistWorkflow(trackIDs []string) {
	go func() {
		pls, err := m.App.ServerManager.Server.GetPlaylists()
		pls = sharedutil.FilterSlice(pls, func(pl *mediaprovider.Playlist) bool {
			return pl.Owner == m.App.ServerManager.LoggedInUser
		})
		if err != nil {
			// TODO: surface this error to user
			log.Printf("error getting user-owned playlists: %s", err.Error())
			return
		}
		plNames := make([]string, 0, len(pls))
		for _, pl := range pls {
			plNames = append(plNames, pl.Name)
		}

		dlg := dialogs.NewAddToPlaylistDialog("Add to Playlist", plNames)
		pop := widget.NewModalPopUp(dlg, m.MainWindow.Canvas())
		m.ClosePopUpOnEscape(pop)
		dlg.OnCanceled = pop.Hide
		dlg.OnSubmit = func(playlistChoice int, newPlaylistName string) {
			pop.Hide()
			m.doModalClosed()
			if playlistChoice < 0 {
				go m.App.ServerManager.Server.CreatePlaylist(newPlaylistName, trackIDs)
			} else {
				go m.App.ServerManager.Server.EditPlaylistTracks(
					pls[playlistChoice].ID, trackIDs, nil /*tracksToRemove*/)
			}
		}
		m.haveModal = true
		pop.Show()
	}()
}

func (m *Controller) DoEditPlaylistWorkflow(playlist *mediaprovider.Playlist) {
	dlg := dialogs.NewEditPlaylistDialog(playlist)
	pop := widget.NewModalPopUp(dlg, m.MainWindow.Canvas())
	m.ClosePopUpOnEscape(pop)
	dlg.OnCanceled = func() {
		pop.Hide()
		m.doModalClosed()
	}
	dlg.OnDeletePlaylist = func() {
		pop.Hide()
		dialog.ShowCustomConfirm("Confirm Delete Playlist", "OK", "Cancel", layout.NewSpacer(), /*custom content*/
			func(ok bool) {
				if !ok {
					pop.Show()
				} else {
					m.doModalClosed()
					go func() {
						if err := m.App.ServerManager.Server.DeletePlaylist(playlist.ID); err != nil {
							log.Printf("error deleting playlist: %s", err.Error())
						} else if rte := m.CurPageFunc(); rte.Page == Playlist && rte.Arg == playlist.ID {
							// navigate to playlists page if user is still on the page of the deleted playlist
							m.NavigateTo(PlaylistsRoute())
						}
					}()
				}
			}, m.MainWindow)
	}
	dlg.OnUpdateMetadata = func() {
		pop.Hide()
		m.doModalClosed()
		go func() {
			err := m.App.ServerManager.Server.EditPlaylist(playlist.ID, dlg.Name, dlg.Description, dlg.IsPublic)
			if err != nil {
				log.Printf("error updating playlist: %s", err.Error())
			} else if rte := m.CurPageFunc(); rte.Page == Playlist && rte.Arg == playlist.ID {
				// if user is on playlist page, reload to get the updates
				m.ReloadFunc()
			}
		}()
	}
	m.haveModal = true
	pop.Show()
}

func (c *Controller) DoConnectToServerWorkflow(server *backend.ServerConfig) {
	pass, err := c.App.ServerManager.GetServerPassword(server.ID)
	if err != nil {
		log.Printf("error getting password from keyring: %v", err)
		c.PromptForLoginAndConnect()
	} else {
		if err := c.tryConnectToServer(server, pass); err != nil {
			dlg := dialog.NewError(err, c.MainWindow)
			dlg.SetOnClosed(func() {
				c.PromptForLoginAndConnect()
			})
			c.haveModal = true
			dlg.Show()
		}
	}
}

func (m *Controller) PromptForLoginAndConnect() {
	// TODO: this will need to be rewritten a bit when we support multi servers
	// need to make sure the intended server is first in the list passed to NewLoginDialog
	d := dialogs.NewLoginDialog(m.App.Config.Servers, m.App.ServerManager.GetServerPassword)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func(server *backend.ServerConfig, password string) {
		d.DisableSubmit()
		d.SetInfoText("Testing connection...")
		go func() {
			err := m.App.ServerManager.TestConnectionAndAuth(server.ServerConnection, password, 5*time.Second)
			if err == backend.ErrUnreachable {
				d.SetErrorText("Server unreachable")
			} else if err != nil {
				d.SetErrorText("Authentication failed")
			} else {
				pop.Hide()
				m.trySetPasswordAndConnectToServer(server, password)
				m.doModalClosed()
			}
			d.EnableSubmit()
		}()
	}
	d.OnEditServer = func(server *backend.ServerConfig) {
		pop.Hide()
		editD := dialogs.NewAddEditServerDialog("Edit server", true, server, m.MainWindow.Canvas().Focus)
		editPop := widget.NewModalPopUp(editD, m.MainWindow.Canvas())
		editD.OnSubmit = func() {
			d.DisableSubmit()
			go func() {
				if m.testConnectionAndUpdateDialogText(editD) {
					// connection is good
					editPop.Hide()
					server.Hostname = editD.Host
					server.AltHostname = editD.AltHost
					server.Nickname = editD.Nickname
					server.Username = editD.Username
					server.LegacyAuth = editD.LegacyAuth
					m.trySetPasswordAndConnectToServer(server, editD.Password)
					m.doModalClosed()
				}
				d.EnableSubmit()
			}()
		}
		editD.OnCancel = func() {
			editPop.Hide()
			pop.Show()
		}
		editPop.Show()
	}
	d.OnNewServer = func() {
		pop.Hide()
		newD := dialogs.NewAddEditServerDialog("Add server", true, nil, m.MainWindow.Canvas().Focus)
		newPop := widget.NewModalPopUp(newD, m.MainWindow.Canvas())
		newD.OnSubmit = func() {
			d.DisableSubmit()
			go func() {
				if m.testConnectionAndUpdateDialogText(newD) {
					// connection is good
					newPop.Hide()
					conn := backend.ServerConnection{
						Hostname:    newD.Host,
						AltHostname: newD.AltHost,
						Username:    newD.Username,
						LegacyAuth:  newD.LegacyAuth,
					}
					server := m.App.Config.AddServer(newD.Nickname, conn)
					m.trySetPasswordAndConnectToServer(server, newD.Password)
					m.doModalClosed()
				}
				d.EnableSubmit()
			}()
		}
		newD.OnCancel = func() {
			newPop.Hide()
			pop.Show()
		}
		newPop.Show()
	}
	d.OnDeleteServer = func(server *backend.ServerConfig) {
		pop.Hide()
		dialog.ShowConfirm("Confirm delete server",
			fmt.Sprintf("Are you sure you want to delete the server %q?", server.Nickname),
			func(ok bool) {
				if ok {
					m.App.Config.DeleteServer(server.ID)
					m.App.DeleteServerCacheDir(server.ID)
					d.SetServers(m.App.Config.Servers)
				}
				if len(m.App.Config.Servers) == 0 {
					m.PromptForFirstServer()
				} else {
					pop.Show()
				}
			}, m.MainWindow)

	}
	m.haveModal = true
	pop.Show()
}

func (c *Controller) ShowAboutDialog() {
	dlg := dialogs.NewAboutDialog(c.AppVersion)
	pop := widget.NewModalPopUp(dlg, c.MainWindow.Canvas())
	dlg.OnDismiss = func() {
		pop.Hide()
		c.doModalClosed()
	}
	c.ClosePopUpOnEscape(pop)
	c.haveModal = true
	pop.Show()
}

func (c *Controller) ShowSettingsDialog(themeUpdateCallbk func()) {
	devs, err := c.App.Player.ListAudioDevices()
	if err != nil {
		log.Printf("error listing audio devices: %v", err)
		devs = []player.AudioDevice{{Name: "auto", Description: "Autoselect device"}}
	}

	dlg := dialogs.NewSettingsDialog(c.App.Config, devs, c.MainWindow)
	dlg.OnReplayGainSettingsChanged = func() {
		c.App.PlaybackManager.SetReplayGainOptions(c.App.Config.ReplayGain)
	}
	dlg.OnAudioExclusiveSettingChanged = func() {
		c.App.Player.SetAudioExclusive(c.App.Config.LocalPlayback.AudioExclusive)
	}
	dlg.OnAudioDeviceSettingChanged = func() {
		c.App.Player.SetAudioDevice(c.App.Config.LocalPlayback.AudioDeviceName)
	}
	dlg.OnThemeSettingChanged = themeUpdateCallbk
	pop := widget.NewModalPopUp(dlg, c.MainWindow.Canvas())
	dlg.OnDismiss = func() {
		pop.Hide()
		c.doModalClosed()
	}
	c.ClosePopUpOnEscape(pop)
	c.haveModal = true
	pop.Show()
}

func (c *Controller) trySetPasswordAndConnectToServer(server *backend.ServerConfig, password string) error {
	if err := c.App.ServerManager.SetServerPassword(server, password); err != nil {
		log.Printf("error setting keyring credentials: %v", err)
		// Don't return an error; fall back to just using the password in-memory
		// User will need to log in with the password on subsequent runs.
	}
	return c.tryConnectToServer(server, password)
}

func (c *Controller) tryConnectToServer(server *backend.ServerConfig, password string) error {
	if err := c.App.ServerManager.TestConnectionAndAuth(server.ServerConnection, password, 10*time.Second); err != nil {
		return err
	}
	if err := c.App.ServerManager.ConnectToServer(server, password); err != nil {
		log.Printf("error connecting to server: %v", err)
		return err
	}
	return nil
}

func (c *Controller) testConnectionAndUpdateDialogText(dlg *dialogs.AddEditServerDialog) bool {
	dlg.SetInfoText("Testing connection...")
	conn := backend.ServerConnection{
		Hostname:    dlg.Host,
		AltHostname: dlg.AltHost,
		Username:    dlg.Username,
		LegacyAuth:  dlg.LegacyAuth,
	}
	err := c.App.ServerManager.TestConnectionAndAuth(conn, dlg.Password, 5*time.Second)
	if err == backend.ErrUnreachable {
		dlg.SetErrorText("Could not reach server (wrong hostname?)")
		return false
	} else if err != nil {
		dlg.SetErrorText("Authentication failed (wrong username/password)")
		return false
	}
	return true
}

func (c *Controller) doModalClosed() {
	c.haveModal = false
	if c.runOnModalClosed != nil {
		c.runOnModalClosed()
		c.runOnModalClosed = nil
	}
}

func (c *Controller) SetTrackFavorites(trackIDs []string, favorite bool) {
	go c.App.ServerManager.Server.SetFavorite(mediaprovider.RatingFavoriteParameters{
		TrackIDs: trackIDs,
	}, favorite)

	for _, id := range trackIDs {
		c.App.PlaybackManager.OnTrackFavoriteStatusChanged(id, favorite)
	}
}

func (c *Controller) SetTrackRatings(trackIDs []string, rating int) {
	go c.App.ServerManager.Server.SetRating(mediaprovider.RatingFavoriteParameters{
		TrackIDs: trackIDs,
	}, rating)

	// Notify PlaybackManager of rating change to update
	// the in-memory track models
	for _, id := range trackIDs {
		c.App.PlaybackManager.OnTrackRatingChanged(id, rating)
	}
}
