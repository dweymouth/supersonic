package browsing

import (
	"supersonic/backend"
	"supersonic/ui/controller"
)

type NavigationHandler interface {
	SetPage(Page)
}

type Router struct {
	App        *backend.App
	Controller *controller.Controller
	Nav        NavigationHandler
}

func NewRouter(app *backend.App, controller *controller.Controller, nav NavigationHandler) Router {
	r := Router{
		App:        app,
		Controller: controller,
		Nav:        nav,
	}
	return r
}

func (r Router) CreatePage(rte controller.Route) Page {
	switch rte.Page {
	case controller.Album:
		return NewAlbumPage(rte.Arg, &r.App.Config.AlbumPage, r.App.ServerManager, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager, r.Controller)
	case controller.Albums:
		return NewAlbumsPage(&r.App.Config.AlbumsPage, r.Controller, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager)
	case controller.Artist:
		return NewArtistPage(rte.Arg, &r.App.Config.ArtistPage, r.App.PlaybackManager, r.App.ServerManager, r.App.ImageManager, r.Controller)
	case controller.Artists:
		return NewArtistsGenresPage(false, r.Controller, r.App.ServerManager)
	case controller.Favorites:
		return NewFavoritesPage(&r.App.Config.FavoritesPage, r.Controller, r.App.ServerManager, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager)
	case controller.Genre:
		return NewGenrePage(rte.Arg, r.Controller, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager)
	case controller.Genres:
		return NewArtistsGenresPage(true, r.Controller, r.App.ServerManager)
	case controller.NowPlaying:
		return NewNowPlayingPage(rte.Arg, r.Controller, &r.App.Config.NowPlayingPage, r.App.ServerManager, r.App.PlaybackManager)
	case controller.Playlist:
		return NewPlaylistPage(rte.Arg, &r.App.Config.PlaylistPage, r.Controller, r.App.ServerManager, r.App.PlaybackManager, r.App.ImageManager)
	case controller.Playlists:
		return NewPlaylistsPage(r.Controller, &r.App.Config.PlaylistsPage, r.App.ServerManager)
	case controller.Tracks:
		return NewTracksPage(r.Controller, &r.App.Config.TracksPage, r.App.LibraryManager)
	}
	return nil
}

func (r Router) NavigateTo(rte controller.Route) {
	r.Nav.SetPage(r.CreatePage(rte))
}
