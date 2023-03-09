package controller

import (
	"image"
	"log"
	"supersonic/backend"
	"supersonic/ui/dialogs"
	"supersonic/ui/util"
	"supersonic/ui/widgets"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic/subsonic"
)

type NavigationHandler func(Route)

type Controller struct {
	// if not nil, this popup should be hidden when escape is pressed
	EscapablePopUp *widget.PopUp

	AppVersion string
	MainWindow fyne.Window
	App        *backend.App
	NavHandler NavigationHandler
}

func (m *Controller) NavigateTo(route Route) {
	m.NavHandler(route)
}

func (m *Controller) ClosePopUpOnEscape(pop *widget.PopUp) {
	m.EscapablePopUp = pop
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
	tracklist.OnAddToQueue = func(tracks []*subsonic.Child) {
		m.App.PlaybackManager.LoadTracks(tracks, true, false)
	}
	tracklist.OnPlayTrackAt = func(idx int) {
		m.App.PlaybackManager.LoadTracks(tracklist.Tracks, false, false)
		m.App.PlaybackManager.PlayTrackAt(idx)
	}
	tracklist.OnPlaySelection = func(tracks []*subsonic.Child) {
		m.App.PlaybackManager.LoadTracks(tracks, false, false)
		m.App.PlaybackManager.PlayFromBeginning()
	}
	tracklist.OnSetFavorite = func(trackIDs []string, fav bool) {
		s := m.App.ServerManager.Server
		if fav {
			go s.Star(subsonic.StarParameters{SongIDs: trackIDs})
		} else {
			go s.Unstar(subsonic.StarParameters{SongIDs: trackIDs})
		}
		for _, id := range trackIDs {
			m.App.PlaybackManager.OnTrackFavoriteStatusChanged(id, fav)
		}
	}
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

func (m *Controller) PromptForFirstServer() {
	d := dialogs.NewAddEditServerDialog("Connect to Server", nil)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func() {
		d.DisableSubmit()
		go func() {
			if m.testConnectionAndUpdateDialogText(d) {
				// connection is good
				pop.Hide()
				server := m.App.Config.AddServer(d.Nickname, d.Host, d.Username, d.LegacyAuth)
				if err := m.App.ServerManager.SetServerPassword(server, d.Password); err != nil {
					log.Printf("error setting keyring credentials: %v", err)
					// TODO: handle?
				}
				m.DoConnectToServerWorkflow(server)
			}
			d.EnableSubmit()
		}()
	}
	pop.Show()
}

// Show dialog to prompt for playlist.
// Depending on the results of that dialog, potentially create a new playlist
// Add tracks to the user-specified playlist
func (m *Controller) DoAddTracksToPlaylistWorkflow(trackIDs []string) {
	pls, err := m.App.LibraryManager.GetUserOwnedPlaylists()
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
		if playlistChoice < 0 {
			m.App.ServerManager.Server.CreatePlaylistWithTracks(
				trackIDs, map[string]string{"name": newPlaylistName})
		} else {
			m.App.ServerManager.Server.UpdatePlaylistTracks(
				pls[playlistChoice].ID, trackIDs, nil /*tracksToRemove*/)
		}
	}
	pop.Show()
}

func (c *Controller) DoConnectToServerWorkflow(server *backend.ServerConfig) {
	pass, err := c.App.ServerManager.GetServerPassword(server)
	if err != nil {
		log.Printf("error getting password from keyring: %v", err)
		c.PromptForLoginAndConnect()
	} else {
		if err := c.tryConnectToServer(server, pass); err != nil {
			dlg := dialog.NewError(err, c.MainWindow)
			dlg.SetOnClosed(func() {
				c.PromptForLoginAndConnect()
			})
			dlg.Show()
		}
	}
}

func (m *Controller) PromptForLoginAndConnect() {
	// TODO: this will need to be rewritten a bit when we support multi servers
	// need to make sure the intended server is first in the list passed to NewLoginDialog
	d := dialogs.NewLoginDialog(m.App.Config.Servers)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func(server *backend.ServerConfig, password string) {
		d.DisableSubmit()
		d.SetInfoText("Testing connection...")
		go func() {
			err := m.App.ServerManager.TestConnectionAndAuth(server.Hostname, server.Username, password, server.LegacyAuth, 5*time.Second)
			if err == backend.ErrUnreachable {
				d.SetErrorText("Server unreachable")
			} else if err != nil {
				d.SetErrorText("Authentication failed")
			} else {
				pop.Hide()
				m.trySetPasswordAndConnectToServer(server, password)
			}
			d.EnableSubmit()
		}()
	}
	d.OnEditServer = func(server *backend.ServerConfig) {
		pop.Hide()
		editD := dialogs.NewAddEditServerDialog("Edit server", server)
		editPop := widget.NewModalPopUp(editD, m.MainWindow.Canvas())
		editD.OnSubmit = func() {
			d.DisableSubmit()
			go func() {
				if m.testConnectionAndUpdateDialogText(editD) {
					// connection is good
					editPop.Hide()
					server.Hostname = editD.Host
					server.Nickname = editD.Nickname
					server.Username = editD.Username
					server.LegacyAuth = editD.LegacyAuth
					m.trySetPasswordAndConnectToServer(server, editD.Password)
				}
				d.EnableSubmit()
			}()
		}
		editPop.Show()
	}
	pop.Show()
}

func (c *Controller) ShowAboutDialog() {
	dlg := dialogs.NewAboutDialog(c.AppVersion)
	pop := widget.NewModalPopUp(dlg, c.MainWindow.Canvas())
	dlg.OnDismiss = pop.Hide
	c.ClosePopUpOnEscape(pop)
	pop.Show()
}

func (c *Controller) trySetPasswordAndConnectToServer(server *backend.ServerConfig, password string) error {
	if err := c.App.ServerManager.SetServerPassword(server, password); err != nil {
		log.Printf("error setting keyring credentials: %v", err)
		// TODO: how best to handle this unexpected codepath
		// fall back to prompting for password on each run of the app?
		return err
	}
	return c.tryConnectToServer(server, password)
}

func (c *Controller) tryConnectToServer(server *backend.ServerConfig, password string) error {
	if err := c.App.ServerManager.TestConnectionAndAuth(server.Hostname, server.Username, password, server.LegacyAuth, 10*time.Second); err != nil {
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
	err := c.App.ServerManager.TestConnectionAndAuth(dlg.Host, dlg.Username, dlg.Password, dlg.LegacyAuth, 5*time.Second)
	if err == backend.ErrUnreachable {
		dlg.SetErrorText("Could not reach server (wrong hostname?)")
		return false
	} else if err != nil {
		dlg.SetErrorText("Authentication failed (wrong username/password)")
		return false
	}
	return true
}
