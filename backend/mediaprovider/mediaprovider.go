package mediaprovider

import "image"

type AlbumFilter struct {
	MinYear int
	MaxYear int      // 0 == unset/match any
	Genres  []string // len(0) == unset/match any

	ExcludeFavorited   bool // mut. exc. with ExcludeUnfavorited
	ExcludeUnfavorited bool // mut. exc. with ExcludeFavorited
}

type AlbumIterator interface {
	Next() *Album
}

type TrackIterator interface {
	Next() *Track
}

type RatingFavoriteParameters struct {
	AlbumIDs  []string
	ArtistIDs []string
	TrackIDs  []string
}

type Favorites struct {
	Albums  []Album
	Artists []Artist
	Tracks  []Track
}

type MediaProvider interface {
	GetAlbum(albumID string) (*AlbumWithTracks, error)

	GetArtist(artistID string) (*Artist, error)

	GetPlaylist(playlistID string) (*PlaylistWithTracks, error)

	GetCoverArt(coverArtID string, size int) (image.Image, error)

	AlbumSortOrders() []string

	IterateAlbums(sortOrder string, searchQuery string, filter AlbumFilter) AlbumIterator

	IterateTracks(searchQuery string) TrackIterator

	GetRandomTracks(genre string, count int) ([]Track, error)

	GetSimilarTracks(artistID string, count int) ([]Track, error)

	GetArtists() ([]Artist, error)

	GetGenres() ([]Genre, error)

	GetFavorites() (Favorites, error)

	GetPlaylists() ([]Playlist, error)

	SetFavorite(params RatingFavoriteParameters) error

	SetRating(params RatingFavoriteParameters) error
}
