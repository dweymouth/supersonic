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
	Albums  []*Album
	Artists []*Artist
	Tracks  []*Track
}

type MediaProvider interface {
	GetAlbum(albumID string) (*AlbumWithTracks, error)

	GetArtist(artistID string) (*ArtistWithAlbums, error)

	GetArtistInfo(artistID string) (*ArtistInfo, error)

	GetPlaylist(playlistID string) (*PlaylistWithTracks, error)

	GetCoverArt(coverArtID string, size int) (image.Image, error)

	AlbumSortOrders() []string

	IterateAlbums(sortOrder string, searchQuery string, filter AlbumFilter) AlbumIterator

	IterateTracks(searchQuery string) TrackIterator

	GetRandomTracks(genre string, count int) ([]*Track, error)

	GetSimilarTracks(artistID string, count int) ([]*Track, error)

	GetArtists() ([]*Artist, error)

	GetGenres() ([]*Genre, error)

	GetFavorites() (Favorites, error)

	GetStreamURL(trackID string) (string, error)

	GetTopTracks(artist Artist, count int) ([]*Track, error)

	SetFavorite(params RatingFavoriteParameters, favorite bool) error

	SetRating(params RatingFavoriteParameters, rating int) error

	GetPlaylists() ([]*Playlist, error)

	CreatePlaylist(name string, trackIDs []string) error

	EditPlaylist(id, name, description string, public bool) error

	EditPlaylistTracks(id string, trackIDsToAdd []string, trackIndexesToRemove []int) error

	ReplacePlaylistTracks(id string, trackIDs []string) error

	DeletePlaylist(id string) error

	Scrobble(trackID string, submission bool) error
}
