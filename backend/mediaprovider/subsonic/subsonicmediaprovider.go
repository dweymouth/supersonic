package subsonic

import (
	"image"
	"log"
	"strconv"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

const (
	AlbumSortRecentlyAdded    string = "Recently Added"
	AlbumSortRecentlyPlayed   string = "Recently Played"
	AlbumSortFrequentlyPlayed string = "Frequently Played"
	AlbumSortRandom           string = "Random"
	AlbumSortTitleAZ          string = "Title (A-Z)"
	AlbumSortArtistAZ         string = "Artist (A-Z)"
	AlbumSortYearAscending    string = "Year (ascending)"
	AlbumSortYearDescending   string = "Year (descending)"
)

type subsonicMediaProvider struct {
	client *subsonic.Client
}

func SubsonicMediaProvider(subsonicClient *subsonic.Client) *subsonicMediaProvider /*mediaprovider.MediaProvider*/ {
	return &subsonicMediaProvider{client: subsonicClient}
}

func (s *subsonicMediaProvider) AlbumSortOrders() []string {
	return []string{
		AlbumSortRecentlyAdded,
		AlbumSortRecentlyPlayed,
		AlbumSortFrequentlyPlayed,
		AlbumSortRandom,
		AlbumSortTitleAZ,
		AlbumSortArtistAZ,
		AlbumSortYearAscending,
		AlbumSortYearDescending,
	}
}

func (s *subsonicMediaProvider) CreatePlaylist(name string, trackIDs []string) error {
	return s.client.CreatePlaylistWithTracks(trackIDs, map[string]string{"name": name})
}

func (s *subsonicMediaProvider) DeletePlaylist(id string) error {
	return s.client.DeletePlaylist(id)
}

func (s *subsonicMediaProvider) EditPlaylist(id, name, description string, public bool) error {
	return s.client.UpdatePlaylist(id, map[string]string{
		"name":    name,
		"comment": description,
		"public":  strconv.FormatBool(public),
	})
}

func (s *subsonicMediaProvider) EditPlaylistTracks(id string, trackIDsToAdd []string, trackIndexesToRemove []int) error {
	return s.client.UpdatePlaylistTracks(id, trackIDsToAdd, trackIndexesToRemove)
}

func (s *subsonicMediaProvider) GetAlbum(albumID string) (*mediaprovider.AlbumWithTracks, error) {
	al, err := s.client.GetAlbum(albumID)
	if err != nil {
		return nil, err
	}
	return &mediaprovider.AlbumWithTracks{
		Album: mediaprovider.Album{
			ID:          al.ID,
			Name:        al.Name,
			ArtistIDs:   []string{al.ArtistID},
			CoverArtID:  al.CoverArt,
			ArtistNames: []string{al.Artist},
			Genres:      []string{al.Genre},
			Year:        al.Year,
			TrackCount:  al.SongCount,
			Favorite:    !al.Starred.IsZero(),
			Duration:    al.Duration,
		},
		Tracks: sharedutil.MapSlice(al.Song, s.toTrack),
	}, nil
}

func (s *subsonicMediaProvider) GetArtist(artistID string) (*mediaprovider.ArtistWithAlbums, error) {
	ar, err := s.client.GetArtist(artistID)
	if err != nil {
		return nil, err
	}
	return &mediaprovider.ArtistWithAlbums{
		Artist: mediaprovider.Artist{
			ID:         ar.ID,
			Name:       ar.Name,
			AlbumCount: ar.AlbumCount,
		},
		Albums: sharedutil.MapSlice(ar.Album, toAlbum),
	}, nil
}

func (s *subsonicMediaProvider) GetArtistInfo(artistID string) (*mediaprovider.ArtistInfo, error) {
	info, err := s.client.GetArtistInfo(artistID, map[string]string{})
	if err != nil {
		return nil, err
	}
	return &mediaprovider.ArtistInfo{
		Biography:      info.Biography,
		LastFMUrl:      info.LastFmUrl,
		ImageURL:       info.LargeImageUrl,
		SimilarArtists: sharedutil.MapSlice(info.SimilarArtist, toArtist),
	}, nil
}

func (s *subsonicMediaProvider) GetArtists() ([]mediaprovider.Artist, error) {
	idxs, err := s.client.GetArtists(map[string]string{})
	if err != nil {
		return nil, err
	}
	var artists []mediaprovider.Artist
	for _, idx := range idxs.Index {
		for _, ar := range idx.Artist {
			artists = append(artists, toArtistFromID3(ar))
		}
	}
	return artists, nil
}

func (s *subsonicMediaProvider) GetCoverArt(id string, size int) (image.Image, error) {
	return s.client.GetCoverArt(id, map[string]string{"size": strconv.Itoa(size)})
}

func (s *subsonicMediaProvider) GetFavorites() (mediaprovider.Favorites, error) {
	fav, err := s.client.GetStarred2(map[string]string{})
	if err != nil {
		return mediaprovider.Favorites{}, err
	}
	return mediaprovider.Favorites{
		Albums:  sharedutil.MapSlice(fav.Album, toAlbum),
		Artists: sharedutil.MapSlice(fav.Artist, toArtistFromID3),
		Tracks:  sharedutil.MapSlice(fav.Song, s.toTrack),
	}, nil
}

func (s *subsonicMediaProvider) GetGenres() ([]mediaprovider.Genre, error) {
	g, err := s.client.GetGenres()
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(g, func(g *subsonic.Genre) mediaprovider.Genre {
		return mediaprovider.Genre{
			Name:       g.Name,
			AlbumCount: g.AlbumCount,
			TrackCount: g.SongCount,
		}
	}), nil
}

func (s *subsonicMediaProvider) GetPlaylist(playlistID string) (*mediaprovider.PlaylistWithTracks, error) {
	pl, err := s.client.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}
	return &mediaprovider.PlaylistWithTracks{
		Playlist: toPlaylist(pl),
		Tracks:   sharedutil.MapSlice(pl.Entry, s.toTrack),
	}, nil
}

func (s *subsonicMediaProvider) GetPlaylists() ([]mediaprovider.Playlist, error) {
	pl, err := s.client.GetPlaylists(map[string]string{})
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(pl, toPlaylist), nil
}

func (s *subsonicMediaProvider) GetRandomTracks(genreName string, count int) ([]mediaprovider.Track, error) {
	opts := map[string]string{"size": strconv.Itoa(count)}
	if genreName != "" {
		opts["genre"] = genreName
	}
	tr, err := s.client.GetRandomSongs(opts)
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(tr, s.toTrack), nil
}

func (s *subsonicMediaProvider) GetSimilarTracks(artistID string, count int) ([]mediaprovider.Track, error) {
	tr, err := s.client.GetSimilarSongs2(artistID, map[string]string{"count": strconv.Itoa(count)})
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(tr, s.toTrack), nil
}

func (s *subsonicMediaProvider) toTrack(ch *subsonic.Child) mediaprovider.Track {
	if ch == nil {
		log.Println("subsonicMediaProvider: toTrack called on nil track")
		return mediaprovider.Track{}
	}
	return mediaprovider.Track{
		ID:          ch.ID,
		CoverArtID:  ch.CoverArt,
		ParentID:    ch.Parent,
		Name:        ch.Title,
		Duration:    ch.Duration,
		TrackNumber: ch.Track,
		DiscNumber:  ch.DiscNumber,
		Genre:       ch.Genre,
		ArtistIDs:   []string{ch.ArtistID},
		ArtistNames: []string{ch.Artist},
		Album:       ch.Album,
		Rating:      ch.UserRating,
		Favorite:    !ch.Starred.IsZero(),
		PlayCount:   int(ch.PlayCount),
		FilePath:    ch.Path,
	}
}

func toAlbum(al *subsonic.AlbumID3) mediaprovider.Album {
	if al == nil {
		log.Println("subsonicMediaProvider: toAlbum called on nil album")
		return mediaprovider.Album{}
	}
	return mediaprovider.Album{
		ID:          al.ID,
		CoverArtID:  al.CoverArt,
		Name:        al.Name,
		Duration:    al.Duration,
		ArtistIDs:   []string{al.ArtistID},
		ArtistNames: []string{al.Artist},
		Year:        al.Year,
		Genres:      []string{al.Genre},
		TrackCount:  al.SongCount,
		Favorite:    !al.Starred.IsZero(),
	}
}

func toArtist(ar *subsonic.Artist) mediaprovider.Artist {
	if ar == nil {
		log.Println("subsonicMediaProvider: toArtist called on nil artist")
		return mediaprovider.Artist{}
	}
	return mediaprovider.Artist{
		ID:   ar.ID,
		Name: ar.Name,
	}
}

func toArtistFromID3(ar *subsonic.ArtistID3) mediaprovider.Artist {
	if ar == nil {
		log.Println("subsonicMediaProvider: toArtistFromID3 called on nil artistID3")
		return mediaprovider.Artist{}
	}
	return mediaprovider.Artist{
		ID:         ar.ID,
		Name:       ar.Name,
		AlbumCount: ar.AlbumCount,
	}
}

func toPlaylist(pl *subsonic.Playlist) mediaprovider.Playlist {
	return mediaprovider.Playlist{
		Name:        pl.Name,
		Description: pl.Comment,
		Owner:       pl.Owner,
		Public:      pl.Public,
		TrackCount:  pl.SongCount,
	}
}
