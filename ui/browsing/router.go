package browsing

import (
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
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
	MainWindow fyne.Window
	Nav        NavigationHandler
}

func NewRouter(app *backend.App, mainWindow fyne.Window, nav NavigationHandler) Router {
	return Router{
		App:        app,
		MainWindow: mainWindow,
		Nav:        nav,
	}
}

type popUpProvider struct {
	window fyne.Window
}

func (p *popUpProvider) CreatePopUp(obj fyne.CanvasObject) *widget.PopUp {
	return widget.NewPopUp(obj, p.window.Canvas())
}

func (p *popUpProvider) WindowSize() fyne.Size {
	return p.window.Canvas().Size()
}

func (r Router) pop() *popUpProvider {
	return &popUpProvider{window: r.MainWindow}
}

func (r Router) CreatePage(rte Route) Page {
	switch rte.Page {
	case Album:
		return NewAlbumPage(rte.Arg, r.App.ServerManager, r.App.LibraryManager, r.App.ImageManager, r.pop(), r.OpenRoute)
	case Albums:
		return NewAlbumsPage("Albums", rte.Arg, r.App.LibraryManager, r.App.ImageManager, r.OpenRoute)
	case Artist:
		return NewArtistPage(rte.Arg, r.App.ServerManager, r.App.ImageManager, r.OpenRoute)
	case Artists:
		return NewArtistsGenresPage(false, r.App.ServerManager, r.OpenRoute)
	case Favorites:
		return NewFavoritesPage(r.App.ServerManager, r.App.ImageManager, r.OpenRoute)
	case Genre:
		return NewGenrePage(rte.Arg, r.App.LibraryManager, r.App.ImageManager, r.OpenRoute)
	case Genres:
		return NewArtistsGenresPage(true, r.App.ServerManager, r.OpenRoute)
	case Playlist:
		return NewPlaylistPage(rte.Arg, r.App.ServerManager, r.App.PlaybackManager, r.OpenRoute)
	case Playlists:
		return NewPlaylistsPage(r.App.ServerManager, r.OpenRoute)
	}
	return nil
}

func (r Router) OpenRoute(rte Route) {
	r.Nav.SetPage(r.CreatePage(rte))
}
