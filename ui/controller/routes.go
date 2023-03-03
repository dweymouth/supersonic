package controller

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
	NowPlaying
	Playlist
	Playlists
)

type Route struct {
	Page PageName
	Arg  string
}

func AlbumsRoute() Route {
	return Route{Page: Albums}
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

func NowPlayingRoute() Route {
	return Route{Page: NowPlaying}
}
