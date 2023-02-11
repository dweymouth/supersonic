package browsing

import (
	"supersonic/backend"
	"supersonic/ui/controller"
	"supersonic/ui/util"
)

type PageName int

const (
	Blank PageName = iota
	Album
	Albums
	Artist
	Artists
	Genre
	Genres
	Favorites
	Playlist
	Playlists
)

type Route struct {
	Page PageName
	Arg  string
}

func AlbumsRoute(sortOrder backend.AlbumSortOrder) Route {
	return Route{Page: Albums, Arg: string(sortOrder)}
}

func ArtistRoute(artistID string) Route {
	return Route{Page: Artist, Arg: artistID}
}

func AlbumRoute(albumID string) Route {
	return Route{Page: Album, Arg: albumID}
}

func FavoritesRoute() Route {
	return Route{Page: Favorites}
}

func GenreRoute(genre string) Route {
	return Route{Page: Genre, Arg: genre}
}

func GenresRoute() Route {
	return Route{Page: Genres}
}

func PlaylistRoute(id string) Route {
	return Route{Page: Playlist, Arg: id}
}
func PlaylistsRoute() Route {
	return Route{Page: Playlists}
}

func ArtistsRoute() Route {
	return Route{Page: Artists}
}

type NavigationHandler interface {
	SetPage(Page)
}

type Router struct {
	App        *backend.App
	Controller *controller.Controller
	Nav        NavigationHandler

	pop util.PopUpProvider
}

func NewRouter(app *backend.App, controller *controller.Controller, nav NavigationHandler) Router {
	r := Router{
		App:        app,
		Controller: controller,
		Nav:        nav,
	}
	return r
}

func (r Router) CreatePage(rte Route) Page {
	switch rte.Page {
	case Album:
		return NewAlbumPage(rte.Arg, r.App.ServerManager, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager, r.Controller, r.OpenRoute)
	case Albums:
		return NewAlbumsPage("Albums", rte.Arg, r.App.PlaybackManager, r.App.LibraryManager, r.App.ImageManager, r.OpenRoute)
	case Artist:
		return NewArtistPage(rte.Arg, r.App.ServerManager, r.App.ImageManager, r.Controller, r.OpenRoute)
	case Artists:
		return NewArtistsGenresPage(false, r.App.ServerManager, r.OpenRoute)
	case Favorites:
		return NewFavoritesPage(r.App.ServerManager, r.App.LibraryManager, r.App.ImageManager, r.OpenRoute)
	case Genre:
		return NewGenrePage(rte.Arg, r.App.LibraryManager, r.App.ImageManager, r.OpenRoute)
	case Genres:
		return NewArtistsGenresPage(true, r.App.ServerManager, r.OpenRoute)
	case Playlist:
		return NewPlaylistPage(rte.Arg, r.Controller, r.App.ServerManager, r.App.PlaybackManager, r.App.ImageManager, r.OpenRoute)
	case Playlists:
		return NewPlaylistsPage(r.App.ServerManager, r.OpenRoute)
	}
	return nil
}

func (r Router) OpenRoute(rte Route) {
	r.Nav.SetPage(r.CreatePage(rte))
}
