package controller

import (
	"archive/zip"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/player"
	"github.com/dweymouth/supersonic/backend/player/mpv"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/dweymouth/supersonic/ui/dialogs"
	myTheme "github.com/dweymouth/supersonic/ui/theme"
	"github.com/dweymouth/supersonic/ui/util"
	"github.com/dweymouth/supersonic/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type NavigationHandler func(Route)

type CurPageFunc func() Route

type ToastProvider interface {
	ShowSuccessToast(string)
	ShowErrorToast(string)
}

type Controller struct {
	visualizationData
	AppVersion string
	App        *backend.App
	MainWindow fyne.Window

	// dependencies injected from MainWindow
	NavHandler          NavigationHandler
	CurPageFunc         CurPageFunc
	ReloadFunc          func()
	RefreshPageFunc     func()
	SelectAllPageFunc   func()
	UnselectAllPageFunc func()
	ToastProvider       ToastProvider

	popUpQueueMutex    sync.Mutex
	popUpQueue         *widget.PopUp
	popUpQueueList     *widgets.PlayQueueList
	popUpQueueLastUsed int64
	escapablePopUp     *widget.PopUp
	haveModal          bool
	runOnModalClosed   func()
}

func New(app *backend.App, appVersion string, mainWindow fyne.Window) *Controller {
	c := &Controller{
		AppVersion: appVersion,
		MainWindow: mainWindow,
		App:        app,
	}
	c.initVisualizations()
	c.App.PlaybackManager.OnQueueChange(func() {
		c.popUpQueueMutex.Lock()
		defer c.popUpQueueMutex.Unlock()
		if c.popUpQueue != nil {
			c.popUpQueueList.SetItems(c.App.PlaybackManager.GetPlayQueue())
		}
	})
	c.App.PlaybackManager.OnSongChange(func(track mediaprovider.MediaItem, _ *mediaprovider.Track) {
		c.popUpQueueMutex.Lock()
		defer c.popUpQueueMutex.Unlock()
		if c.popUpQueue == nil {
			return
		}
		if track == nil {
			c.popUpQueueList.SetNowPlaying("")
		} else {
			c.popUpQueueList.SetNowPlaying(track.Metadata().ID)
		}
	})
	return c
}

func (m *Controller) SelectAll() {
	m.popUpQueueMutex.Lock()
	if m.popUpQueue != nil && m.popUpQueue.Visible() {
		m.popUpQueueList.SelectAll()
		m.popUpQueueMutex.Unlock()
		return
	}
	m.popUpQueueMutex.Unlock()
	if m.SelectAllPageFunc != nil {
		m.SelectAllPageFunc()
	}
}

func (m *Controller) UnselectAll() {
	m.popUpQueueMutex.Lock()
	if m.popUpQueue != nil && m.popUpQueue.Visible() {
		m.popUpQueueList.UnselectAll()
		m.popUpQueueMutex.Unlock()
		return
	}
	m.popUpQueueMutex.Unlock()
	if m.SelectAllPageFunc != nil {
		m.UnselectAllPageFunc()
	}
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

func (m *Controller) HaveModal() bool {
	return m.haveModal
}

func (m *Controller) ShowPopUpPlayQueue() {
	m.popUpQueueMutex.Lock()
	if m.popUpQueue == nil {
		m.popUpQueueList = widgets.NewPlayQueueList(m.App.ImageManager, false)
		m.popUpQueueList.Reorderable = true
		m.popUpQueueList.SetItems(m.App.PlaybackManager.GetPlayQueue())
		m.ConnectPlayQueuelistActions(m.popUpQueueList)

		title := widget.NewRichTextWithText(lang.L("Play Queue"))
		title.Segments[0].(*widget.TextSegment).Style.Alignment = fyne.TextAlignCenter
		title.Segments[0].(*widget.TextSegment).Style.TextStyle.Bold = true
		ctr := container.NewBorder(title, nil, nil, nil,
			container.NewPadded(m.popUpQueueList),
		)
		m.popUpQueue = widget.NewPopUp(ctr, m.MainWindow.Canvas())
		fynetooltip.AddPopUpToolTipLayer(m.popUpQueue)

		container.NewThemeOverride(m.popUpQueue, myTheme.WithColorTransformOverride(
			theme.ColorNameOverlayBackground,
			func(c color.Color) color.Color {
				if nrgba, ok := c.(color.NRGBA); ok {
					nrgba.A = 245
					return nrgba
				}
				c_ := c.(color.RGBA)
				c_.A = 245
				return c_
			},
		))

		// free popUpQueue if it hasn't been used in awhile
		go func() {
			t := time.NewTicker(1 * time.Minute)
			for range t.C {
				m.popUpQueueMutex.Lock()
				now := time.Now().UnixMilli()
				if m.popUpQueueLastUsed < now-120_000 /*2 min*/ {
					fynetooltip.DestroyPopUpToolTipLayer(m.popUpQueue)
					m.popUpQueue = nil
					m.popUpQueueList = nil
					m.popUpQueueLastUsed = 0
					m.popUpQueueMutex.Unlock()
					t.Stop()
					return
				}
				m.popUpQueueMutex.Unlock()
			}
		}()
	}
	m.popUpQueueLastUsed = time.Now().UnixMilli()
	popUpQueueList := m.popUpQueueList
	pop := m.popUpQueue
	m.popUpQueueMutex.Unlock()

	npID := ""
	if np := m.App.PlaybackManager.NowPlaying(); np != nil {
		npID = np.Metadata().ID
	}
	popUpQueueList.SetNowPlaying(npID)
	popUpQueueList.UnselectAll()

	m.ClosePopUpOnEscape(pop)
	minSize := fyne.NewSize(300, 400)
	maxSize := fyne.NewSize(800, 1000)
	canvasSize := m.MainWindow.Canvas().Size()
	size := minSize.Max(maxSize.Min(
		fyne.NewSize(canvasSize.Width*0.4, canvasSize.Height*0.5),
	))
	pop.Resize(size)
	popUpQueueList.ScrollToNowPlaying() // must come after resize
	pop.ShowAtPosition(fyne.NewPos(
		canvasSize.Width-size.Width-10,
		canvasSize.Height-size.Height-100,
	))
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

func (m *Controller) GetArtistTracks(artistID string) []*mediaprovider.Track {
	if mp := m.App.ServerManager.Server; mp != nil {
		if tr, err := mp.GetArtistTracks(artistID); err != nil {
			log.Println(err.Error())
			return nil
		} else {
			return tr
		}
	}
	return nil
}

func (m *Controller) PromptForFirstServer() {
	d := dialogs.NewAddEditServerDialog(lang.L("Connect to Server"), false, nil, m.MainWindow.Canvas().Focus)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func() {
		d.DisableSubmit()
		go func() {
			if m.testConnectionAndUpdateDialogText(d) {
				// connection is good
				pop.Hide()
				m.doModalClosed()
				conn := backend.ServerConnection{
					ServerType:  d.ServerType,
					Hostname:    d.Host,
					AltHostname: d.AltHost,
					Username:    d.Username,
					LegacyAuth:  d.LegacyAuth,
				}
				server := m.App.ServerManager.AddServer(d.Nickname, conn)
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

// Show dialog to select playlist.
// Depending on the results of that dialog, potentially create a new playlist
// Add tracks to the user-specified playlist
func (m *Controller) DoAddTracksToPlaylistWorkflow(trackIDs []string) {
	sp := dialogs.NewSelectPlaylistDialog(m.App.ServerManager.Server, m.App.ImageManager,
		m.App.ServerManager.LoggedInUser, m.App.Config.Application.AddToPlaylistSkipDuplicates)
	pop := widget.NewModalPopUp(sp.SearchDialog, m.MainWindow.Canvas())
	sp.SetOnDismiss(func() {
		pop.Hide()
		m.doModalClosed()
	})
	sp.SetOnNavigateTo(func(contentType mediaprovider.ContentType, id string) {
		notifySuccess := func(n int) {
			fyne.Do(func() {
				msg := lang.LocalizePluralKey("playlist.addedtracks",
					"Added tracks to playlist", n, map[string]string{"trackCount": strconv.Itoa(n)})
				m.ToastProvider.ShowSuccessToast(msg)
			})
		}
		notifyError := func() {
			fyne.Do(func() {
				m.ToastProvider.ShowErrorToast(
					lang.L("An error occurred adding tracks to the playlist"),
				)
			})
		}
		pop.Hide()
		m.App.Config.Application.AddToPlaylistSkipDuplicates = sp.SkipDuplicates
		if id == "" /* creating new playlist */ {
			go func() {
				err := m.App.ServerManager.Server.CreatePlaylist(sp.SearchDialog.SearchQuery(), trackIDs)
				if err == nil {
					notifySuccess(len(trackIDs))
				} else {
					log.Printf("error adding tracks to playlist: %s", err.Error())
					notifyError()
				}
			}()
		} else {
			m.App.Config.Application.DefaultPlaylistID = id
			if sp.SkipDuplicates {
				go func() {
					currentTrackIDs := make(map[string]struct{})
					if selectedPlaylist, err := m.App.ServerManager.Server.GetPlaylist(id); err != nil {
						log.Printf("error getting playlist: %s", err.Error())
						notifyError()
					} else {
						for _, track := range selectedPlaylist.Tracks {
							currentTrackIDs[track.ID] = struct{}{}
						}
						filterTrackIDs := sharedutil.FilterSlice(trackIDs, func(trackID string) bool {
							_, ok := currentTrackIDs[trackID]
							return !ok
						})
						err := m.App.ServerManager.Server.AddPlaylistTracks(id, filterTrackIDs)
						if err == nil {
							notifySuccess(len(filterTrackIDs))
						} else {
							log.Printf("error adding tracks to playlist: %s", err.Error())
							notifyError()
						}
					}
				}()
			} else {
				go func() {
					err := m.App.ServerManager.Server.AddPlaylistTracks(id, trackIDs)
					if err == nil {
						notifySuccess(len(trackIDs))
					} else {
						log.Printf("error adding tracks to playlist: %s", err.Error())
						notifyError()
					}
				}()
			}
		}

	})
	m.ClosePopUpOnEscape(pop)
	m.haveModal = true
	min := sp.MinSize()
	height := fyne.Max(min.Height, fyne.Min(min.Height*1.5, m.MainWindow.Canvas().Size().Height*0.7))
	sp.SearchDialog.Show()
	pop.Resize(fyne.NewSize(min.Width, height))
	pop.Show()
	m.MainWindow.Canvas().Focus(sp.GetSearchEntry())
}

func (m *Controller) DoEditPlaylistWorkflow(playlist *mediaprovider.Playlist) {
	canMakePublic := m.App.ServerManager.Server.CanMakePublicPlaylist()
	dlg := dialogs.NewEditPlaylistDialog(playlist, canMakePublic)
	pop := widget.NewModalPopUp(dlg, m.MainWindow.Canvas())
	m.ClosePopUpOnEscape(pop)
	dlg.OnCanceled = func() {
		pop.Hide()
		m.doModalClosed()
	}
	dlg.OnDeletePlaylist = func() {
		pop.Hide()
		dialog.ShowCustomConfirm(lang.L("Confirm Delete Playlist"), lang.L("OK"), lang.L("Cancel"), layout.NewSpacer(), /*custom content*/
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
							fyne.Do(func() { m.NavigateTo(PlaylistsRoute()) })
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
				fyne.Do(m.ReloadFunc)
			}
		}()
	}
	m.haveModal = true
	pop.Show()
}

// DoConnectToServerWorkflow does the workflow for connecting to the last active server on startup
func (c *Controller) DoConnectToServerWorkflow(server *backend.ServerConfig) {
	pass, err := c.App.ServerManager.GetServerPassword(server.ID)
	if err != nil {
		log.Printf("error getting password from keyring: %v", err)
		c.PromptForLoginAndConnect()
		return
	}

	// try connecting to last used server - set up cancelable modal dialog
	canceled := false
	ctx, cancel := context.WithCancel(context.Background())
	dlg := dialog.NewCustom(lang.L("Connecting"), lang.L("Cancel"),
		widget.NewLabel(fmt.Sprintf(lang.L("Connecting to")+" %s", server.Nickname)), c.MainWindow)
	dlg.SetOnClosed(func() {
		canceled = true
		cancel()
	})
	c.haveModal = true
	dlg.Show()

	// try to connect
	go func() {
		defer cancel() // make sure to free up ctx resources if user does not cancel

		if err := c.tryConnectToServer(ctx, server, pass); err != nil {
			fyne.Do(func() {
				dlg.Hide()
				c.haveModal = false
				if canceled {
					c.PromptForLoginAndConnect()
				} else {
					// connection failure
					dlg := dialog.NewError(err, c.MainWindow)
					dlg.SetOnClosed(func() {
						c.PromptForLoginAndConnect()
					})
					c.haveModal = true
					dlg.Show()
				}
			})
		} else {
			fyne.Do(func() {
				dlg.Hide()
				c.haveModal = false
			})
		}
	}()
}

func (m *Controller) PromptForLoginAndConnect() {
	d := dialogs.NewLoginDialog(m.App.Config.Servers, m.App.ServerManager.GetServerPassword)
	pop := widget.NewModalPopUp(d, m.MainWindow.Canvas())
	d.OnSubmit = func(server *backend.ServerConfig, password string) {
		d.DisableSubmit()
		d.SetInfoText(lang.L("Testing connection") + "...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := m.App.ServerManager.TestConnectionAndAuth(ctx, server.ServerConnection, password)
			fyne.Do(func() {
				if err == backend.ErrUnreachable {
					d.SetErrorText(lang.L("Server unreachable"))
				} else if err != nil {
					d.SetErrorText(lang.L("Authentication failed"))
				} else {
					pop.Hide()
					m.trySetPasswordAndConnectToServer(server, password)
					m.doModalClosed()
				}
				d.EnableSubmit()
			})
		}()
	}
	d.OnEditServer = func(server *backend.ServerConfig) {
		pop.Hide()
		editD := dialogs.NewAddEditServerDialog(lang.L("Edit server"), true, server, m.MainWindow.Canvas().Focus)
		editPop := widget.NewModalPopUp(editD, m.MainWindow.Canvas())
		editD.OnSubmit = func() {
			d.DisableSubmit()
			go func() {
				success := m.testConnectionAndUpdateDialogText(editD)
				fyne.Do(func() {
					if success {
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
				})
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
		newD := dialogs.NewAddEditServerDialog(lang.L("Add Server"), true, nil, m.MainWindow.Canvas().Focus)
		newPop := widget.NewModalPopUp(newD, m.MainWindow.Canvas())
		newD.OnSubmit = func() {
			d.DisableSubmit()
			go func() {
				success := m.testConnectionAndUpdateDialogText(newD)
				fyne.Do(func() {
					if success {
						// connection is good
						newPop.Hide()
						conn := backend.ServerConnection{
							ServerType:  newD.ServerType,
							Hostname:    newD.Host,
							AltHostname: newD.AltHost,
							Username:    newD.Username,
							LegacyAuth:  newD.LegacyAuth,
						}
						server := m.App.ServerManager.AddServer(newD.Nickname, conn)
						m.trySetPasswordAndConnectToServer(server, newD.Password)
						m.doModalClosed()
					}
					d.EnableSubmit()
				})
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
		dialog.ShowConfirm(lang.L("Confirm Delete Server"),
			fmt.Sprintf(lang.L("Are you sure you want to delete the server")+" %q?", server.Nickname),
			func(ok bool) {
				if ok {
					m.App.ServerManager.DeleteServer(server.ID)
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

func (c *Controller) ShowSettingsDialog(themeUpdateCallbk func(), themeFiles map[string]string) {
	devs, err := c.App.LocalPlayer.ListAudioDevices()
	if err != nil {
		log.Printf("error listing audio devices: %v", err)
		devs = []mpv.AudioDevice{{Name: "auto", Description: lang.L("Autoselect device")}}
	}

	curPlayer := c.App.PlaybackManager.CurrentPlayer()
	_, isReplayGainPlayer := curPlayer.(player.ReplayGainPlayer)
	_, isEqualizerPlayer := curPlayer.(*mpv.Player)
	_, canSavePlayQueue := c.App.ServerManager.Server.(mediaprovider.CanSavePlayQueue)
	isLocalPlayer := isEqualizerPlayer
	bands := c.App.LocalPlayer.Equalizer().BandFrequencies()
	dlg := dialogs.NewSettingsDialog(c.App.Config,
		devs, themeFiles, bands,
		c.App.ServerManager.Server.ClientDecidesScrobble(),
		isLocalPlayer, isReplayGainPlayer, isEqualizerPlayer, canSavePlayQueue,
		c.MainWindow)
	dlg.OnReplayGainSettingsChanged = func() {
		c.App.PlaybackManager.SetReplayGainOptions(c.App.Config.ReplayGain)
	}
	dlg.OnAudioExclusiveSettingChanged = func() {
		c.App.LocalPlayer.SetAudioExclusive(c.App.Config.LocalPlayback.AudioExclusive)
	}
	dlg.OnAudioDeviceSettingChanged = func() {
		c.App.LocalPlayer.SetAudioDevice(c.App.Config.LocalPlayback.AudioDeviceName)
	}
	dlg.OnThemeSettingChanged = themeUpdateCallbk
	dlg.OnEqualizerSettingsChanged = func() {
		// currently we only have one equalizer type
		eq := c.App.LocalPlayer.Equalizer().(*mpv.ISO15BandEqualizer)
		eq.Disabled = !c.App.Config.LocalPlayback.EqualizerEnabled
		eq.EQPreamp = c.App.Config.LocalPlayback.EqualizerPreamp
		copy(eq.BandGains[:], c.App.Config.LocalPlayback.GraphicEqualizerBands)
		c.App.LocalPlayer.SetEqualizer(eq)
	}
	dlg.OnPageNeedsRefresh = c.RefreshPageFunc
	pop := widget.NewModalPopUp(dlg, c.MainWindow.Canvas())
	fynetooltip.AddPopUpToolTipLayer(pop)
	dlg.OnDismiss = func() {
		pop.Hide()
		fynetooltip.DestroyPopUpToolTipLayer(pop)
		c.doModalClosed()
		c.App.SaveConfigFile()
	}
	c.ClosePopUpOnEscape(pop)
	c.haveModal = true
	pop.Show()
}

func (c *Controller) ShowQuickSearch() {
	qs := dialogs.NewQuickSearch(c.App.ServerManager.Server, c.App.ImageManager)
	pop := widget.NewModalPopUp(qs.SearchDialog, c.MainWindow.Canvas())
	qs.SetOnDismiss(func() {
		pop.Hide()
		c.doModalClosed()
	})
	qs.SetOnNavigateTo(func(contentType mediaprovider.ContentType, id string) {
		pop.Hide()
		c.doModalClosed()
		switch contentType {
		case mediaprovider.ContentTypeAlbum:
			c.NavigateTo(AlbumRoute(id))
		case mediaprovider.ContentTypeArtist:
			c.NavigateTo(ArtistRoute(id))
		case mediaprovider.ContentTypeTrack:
			go c.App.PlaybackManager.PlayTrack(id)
		case mediaprovider.ContentTypePlaylist:
			c.NavigateTo(PlaylistRoute(id))
		case mediaprovider.ContentTypeGenre:
			c.NavigateTo(GenreRoute(id))
		case mediaprovider.ContentTypeRadioStation:
			if rp, ok := c.App.ServerManager.Server.(mediaprovider.RadioProvider); ok {
				if radio, err := rp.GetRadioStation(id); err == nil {
					go c.App.PlaybackManager.PlayRadioStation(radio)
				}
			}
		}
	})
	c.ClosePopUpOnEscape(pop)
	c.haveModal = true
	min := qs.MinSize()
	height := fyne.Max(min.Height, fyne.Min(min.Height*1.5, c.MainWindow.Canvas().Size().Height*0.7))
	qs.SearchDialog.Show()
	pop.Resize(fyne.NewSize(min.Width, height))
	pop.Show()
	c.MainWindow.Canvas().Focus(qs.GetSearchEntry())
}

func (c *Controller) trySetPasswordAndConnectToServer(server *backend.ServerConfig, password string) error {
	if err := c.App.ServerManager.SetServerPassword(server, password); err != nil {
		log.Printf("error setting keyring credentials: %v", err)
		// Don't return an error; fall back to just using the password in-memory
		// User will need to log in with the password on subsequent runs.
	}
	return c.tryConnectToServer(context.Background(), server, password)
}

// try to connect to the given server, with a 10 second timeout added to the context
func (c *Controller) tryConnectToServer(ctx context.Context, server *backend.ServerConfig, password string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.App.ServerManager.TestConnectionAndAuth(ctx, server.ServerConnection, password); err != nil {
		return err
	}
	if err := c.App.ServerManager.ConnectToServer(server, password); err != nil {
		log.Printf("error connecting to server: %v", err)
		return err
	}
	return nil
}

func (c *Controller) testConnectionAndUpdateDialogText(dlg *dialogs.AddEditServerDialog) bool {
	dlg.SetInfoText(lang.L("Testing connection") + "...")
	conn := backend.ServerConnection{
		ServerType:  dlg.ServerType,
		Hostname:    dlg.Host,
		AltHostname: dlg.AltHost,
		Username:    dlg.Username,
		LegacyAuth:  dlg.LegacyAuth,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.App.ServerManager.TestConnectionAndAuth(ctx, conn, dlg.Password)
	if err == backend.ErrUnreachable {
		dlg.SetErrorText(lang.L("Could not reach server") + fmt.Sprintf(" (%s?)", lang.L("wrong URL")))
		return false
	} else if err != nil {
		dlg.SetErrorText(lang.L("Authentication failed") + fmt.Sprintf(" (%s)", lang.L("wrong username/password")))
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
	r, ok := c.App.ServerManager.Server.(mediaprovider.SupportsRating)
	if !ok {
		return
	}
	go r.SetRating(mediaprovider.RatingFavoriteParameters{
		TrackIDs: trackIDs,
	}, rating)

	// Notify PlaybackManager of rating change to update
	// the in-memory track models
	for _, id := range trackIDs {
		c.App.PlaybackManager.OnTrackRatingChanged(id, rating)
	}
}

func (c *Controller) ShowShareDialog(id string) {
	go func() {
		shareUrl, err := c.createShareURL(id)
		if err != nil {
			return
		}

		fyne.Do(func() {
			hyperlink := widget.NewHyperlink(shareUrl.String(), shareUrl)
			dlg := dialog.NewCustom(lang.L("Share content"), lang.L("OK"),
				container.NewHBox(
					hyperlink,
					widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
						c.MainWindow.Clipboard().SetContent(hyperlink.Text)
					}),
					widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
						if shareUrl, err := c.createShareURL(id); err == nil {
							hyperlink.Text = shareUrl.String()
							hyperlink.URL = shareUrl
							hyperlink.Refresh()
						}
					}),
				),
				c.MainWindow,
			)
			dlg.Show()
		})
	}()
}

func (c *Controller) createShareURL(id string) (*url.URL, error) {
	r, ok := c.App.ServerManager.Server.(mediaprovider.SupportsSharing)
	if !ok {
		return nil, fmt.Errorf("server does not support sharing")
	}

	shareUrl, err := r.CreateShareURL(id)
	if err != nil {
		log.Printf("error creating share URL: %v", err)
		c.showError(
			"Failed to share content. This commonly occurs when the server does not support sharing, " +
				"or has the feature disabled.\nPlease check the server's settings and try again.",
		)
		return nil, err
	}
	return shareUrl, nil
}

func (c *Controller) ShowDownloadDialog(tracks []*mediaprovider.Track, downloadName string) {
	numTracks := len(tracks)
	var fileName string
	if numTracks == 1 {
		fileName = filepath.Base(tracks[0].FilePath)
	} else {
		fileName = "downloaded_tracks.zip"
	}

	dg := dialog.NewFileSave(
		func(file fyne.URIWriteCloser, err error) {
			if err != nil {
				log.Println(err)
				return
			}

			if file == nil {
				return
			}
			if numTracks == 1 {
				go c.downloadTrack(tracks[0], file.URI().Path())
			} else {
				go c.downloadTracks(tracks, file.URI().Path(), downloadName)
			}

		},
		c.MainWindow)
	dg.SetFileName(fileName)
	dg.Show()
}

func (c *Controller) downloadTrack(track *mediaprovider.Track, filePath string) {
	reader, err := c.App.ServerManager.Server.DownloadTrack(track.ID)
	if err != nil {
		log.Println(err)
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Saved song %s to: %s\n", track.Title, filePath)
	c.sendNotification(fmt.Sprintf(lang.L("Download completed")+": %s", track.Title), fmt.Sprintf(lang.L("Saved at")+": %s", filePath))
}

func (c *Controller) downloadTracks(tracks []*mediaprovider.Track, filePath, downloadName string) {
	zipFile, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, track := range tracks {
		reader, err := c.App.ServerManager.Server.DownloadTrack(track.ID)
		if err != nil {
			log.Println(err)
			continue
		}

		fileName := filepath.Base(track.FilePath)

		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = io.Copy(fileWriter, reader)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Saved song %s to: %s\n", track.Title, filePath)
	}

	c.sendNotification(fmt.Sprintf(lang.L("Download completed")+": %s", downloadName), fmt.Sprintf("Saved at: %s", filePath))
}

func (c *Controller) sendNotification(title, content string) {
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   title,
		Content: content,
	})
}

func (c *Controller) showError(content string) {
	// TODO: display an in-app toast message instead of a dialog.
	dialog.ShowError(fmt.Errorf(content), c.MainWindow)
}

func (c *Controller) ShowAlbumInfoDialog(albumID, albumName string, albumCover image.Image) {
	go func() {
		albumInfo, err := c.App.ServerManager.Server.GetAlbumInfo(albumID)
		if err != nil {
			log.Print("Error getting album info: ", err)
			return
		}
		fyne.Do(func() {
			dlg := dialogs.NewAlbumInfoDialog(albumInfo, albumName, albumCover)
			pop := widget.NewModalPopUp(dlg, c.MainWindow.Canvas())
			dlg.OnDismiss = func() {
				pop.Hide()
				c.doModalClosed()
			}
			c.ClosePopUpOnEscape(pop)
			c.haveModal = true
			pop.Show()
		})
	}()
}

func (c *Controller) ShowTrackInfoDialog(track *mediaprovider.Track) {
	info := dialogs.NewTrackInfoDialog(track)
	pop := widget.NewModalPopUp(info, c.MainWindow.Canvas())
	info.OnDismiss = func() {
		pop.Hide()
		c.doModalClosed()
	}
	info.OnNavigateToAlbum = func(albumID string) {
		info.OnDismiss()
		c.NavigateTo(AlbumRoute(albumID))
	}
	info.OnNavigateToArtist = func(artistID string) {
		info.OnDismiss()
		c.NavigateTo(ArtistRoute(artistID))
	}
	info.OnNavigateToGenre = func(genre string) {
		info.OnDismiss()
		c.NavigateTo(GenreRoute(genre))
	}
	info.OnCopyFilePath = func() {
		c.MainWindow.Clipboard().SetContent(track.FilePath)
	}
	c.ClosePopUpOnEscape(pop)
	winSize := c.MainWindow.Canvas().Size()
	popMin := pop.MinSize()
	width := fyne.Min(750, fyne.Max(popMin.Width, winSize.Width*0.8))
	height := fyne.Min(650, fyne.Max(popMin.Height, winSize.Height*0.8))
	pop.Resize(fyne.NewSize(width, height))
	c.haveModal = true
	pop.Show()
}

func (c *Controller) GetSongRadioTracks(sourceTrack *mediaprovider.Track) ([]*mediaprovider.Track, error) {
	radioTracks, err := c.App.ServerManager.Server.GetSongRadio(sourceTrack.ID, 100)
	if err != nil {
		return nil, fmt.Errorf("error getting song radio: %s", err.Error())
	}

	// The goal of this implementation is to place the source track first in the queue.
	filteredTracks := sharedutil.FilterSlice(radioTracks, func(track *mediaprovider.Track) bool {
		return track.ID != sourceTrack.ID
	})
	tracks := []*mediaprovider.Track{sourceTrack}
	tracks = append(tracks, filteredTracks...)
	return tracks, nil
}
