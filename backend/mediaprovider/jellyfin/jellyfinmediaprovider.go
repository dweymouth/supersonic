package jellyfin

import (
	"errors"
	"image"
	"io"
	"math"
	"sync"
	"time"

	"github.com/dweymouth/go-jellyfin"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

const cacheValidDurationSeconds = 60

var _ mediaprovider.MediaProvider = (*jellyfinMediaProvider)(nil)

type jellyfinMediaProvider struct {
	client          *jellyfin.Client
	prefetchCoverCB func(coverArtID string)

	genresCached   []*mediaprovider.Genre
	genresCachedAt int64 // unix
}

func newJellyfinMediaProvider(cli *jellyfin.Client) mediaprovider.MediaProvider {
	return &jellyfinMediaProvider{
		client:       cli,
		genresCached: make([]*mediaprovider.Genre, 0),
	}
}

func (j *jellyfinMediaProvider) SetPrefetchCoverCallback(cb func(coverArtID string)) {
	j.prefetchCoverCB = cb
}

func (jellyfinMediaProvider) CreatePlaylist(name string, trackIDs []string) error {
	return errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) DeletePlaylist(id string) error {
	return j.client.DeletePlaylist(id)
}

func (s *jellyfinMediaProvider) EditPlaylist(id, name, description string, public bool) error {
	return errors.New("unimplemented")
}

func (s *jellyfinMediaProvider) EditPlaylistTracks(id string, trackIDsToAdd []string, trackIndexesToRemove []int) error {
	return errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) GetAlbum(albumID string) (*mediaprovider.AlbumWithTracks, error) {
	al, err := j.client.GetAlbum(albumID)
	if err != nil {
		return nil, err
	}
	var opts jellyfin.QueryOpts
	opts.Filter.ParentID = albumID
	tr, err := j.client.GetSongs(opts)
	if err != nil {
		return nil, err
	}

	album := &mediaprovider.AlbumWithTracks{}
	fillAlbum(al, &album.Album)
	album.Tracks = sharedutil.MapSlice(tr, toTrack)
	return album, nil
}

func (s *jellyfinMediaProvider) GetAlbumInfo(albumID string) (*mediaprovider.AlbumInfo, error) {
	return nil, errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) GetArtist(artistID string) (*mediaprovider.ArtistWithAlbums, error) {
	ar, err := j.client.GetArtist(artistID)
	if err != nil {
		return nil, err
	}
	var opts jellyfin.QueryOpts
	opts.Filter.ArtistID = artistID
	al, err := j.client.GetAlbums(opts)
	if err != nil {
		return nil, err
	}

	artist := &mediaprovider.ArtistWithAlbums{
		Albums: sharedutil.MapSlice(al, toAlbum),
	}
	fillArtist(ar, &artist.Artist)
	return artist, nil
}

func (j *jellyfinMediaProvider) GetArtistInfo(artistID string) (*mediaprovider.ArtistInfo, error) {
	ar, err := j.client.GetArtist(artistID)
	if err != nil {
		return nil, err
	}
	similar, err := j.client.GetSimilarArtists(artistID)
	if err != nil {
		return nil, err
	}
	return &mediaprovider.ArtistInfo{
		SimilarArtists: sharedutil.MapSlice(similar, toArtist),
		Biography:      ar.Overview,
	}, nil
}

func (j *jellyfinMediaProvider) GetArtists() ([]*mediaprovider.Artist, error) {
	ar, err := j.client.GetAlbumArtists(jellyfin.QueryOpts{})
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(ar, toArtist), nil
}

func (j *jellyfinMediaProvider) GetTrack(trackID string) (*mediaprovider.Track, error) {
	tr, err := j.client.GetSong(trackID)
	if err != nil {
		return nil, err
	}
	return toTrack(tr), nil
}

func (j *jellyfinMediaProvider) GetTopTracks(artist mediaprovider.Artist, limit int) ([]*mediaprovider.Track, error) {
	var opts jellyfin.QueryOpts
	opts.Paging.Limit = limit
	opts.Filter.ArtistID = artist.ID
	opts.Sort.Field = "CommunityRating"
	opts.Sort.Mode = jellyfin.SortDesc
	tr, err := j.client.GetSongs(opts)
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(tr, toTrack), nil
}

func (j *jellyfinMediaProvider) GetRandomTracks(genreName string, limit int) ([]*mediaprovider.Track, error) {
	var opts jellyfin.QueryOpts
	opts.Paging.Limit = limit
	opts.Filter.Genres = []jellyfin.NameID{{Name: genreName}}
	opts.Sort.Field = "Random"
	tr, err := j.client.GetSongs(opts)
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(tr, toTrack), nil
}

func (j *jellyfinMediaProvider) GetSimilarTracks(artistID string, limit int) ([]*mediaprovider.Track, error) {
	tr, err := j.client.GetSimilarSongs(artistID, limit)
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(tr, toTrack), nil
}

func (j *jellyfinMediaProvider) GetCoverArt(id string, size int) (image.Image, error) {
	return j.client.GetItemImage(id, "Primary", size, 92)
}

func (s *jellyfinMediaProvider) GetFavorites() (mediaprovider.Favorites, error) {
	return mediaprovider.Favorites{}, errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) GetGenres() ([]*mediaprovider.Genre, error) {
	if j.genresCached != nil && time.Now().Unix()-j.genresCachedAt < cacheValidDurationSeconds {
		return j.genresCached, nil
	}

	g, err := j.client.GetGenres(jellyfin.Paging{})
	if err != nil {
		return nil, err
	}
	j.genresCached = sharedutil.MapSlice(g, func(g jellyfin.NameID) *mediaprovider.Genre {
		return &mediaprovider.Genre{
			Name:       g.Name,
			AlbumCount: -1, // unsupported by Jellyfin
			TrackCount: -1, // unsupported by Jellyfin
		}
	})
	j.genresCachedAt = time.Now().Unix()
	return j.genresCached, nil
}

func (j *jellyfinMediaProvider) GetPlaylists() ([]*mediaprovider.Playlist, error) {
	pl, err := j.client.GetPlaylists()
	if err != nil {
		return nil, err
	}
	return sharedutil.MapSlice(pl, toPlaylist), nil
}

func (j *jellyfinMediaProvider) GetPlaylist(playlistID string) (*mediaprovider.PlaylistWithTracks, error) {
	tr, err := j.client.GetPlaylistSongs(playlistID)
	if err != nil {
		return nil, err
	}
	pl, err := j.client.GetPlaylist(playlistID)
	if err != nil {
		return nil, err
	}

	playlist := &mediaprovider.PlaylistWithTracks{
		Tracks: sharedutil.MapSlice(tr, toTrack),
	}
	fillPlaylist(pl, &playlist.Playlist)
	return playlist, nil
}

func (s *jellyfinMediaProvider) ReplacePlaylistTracks(playlistID string, trackIDs []string) error {
	return errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) SetFavorite(params mediaprovider.RatingFavoriteParameters, favorite bool) error {
	var allIDs []string
	allIDs = append(allIDs, params.AlbumIDs...)
	allIDs = append(allIDs, params.ArtistIDs...)
	allIDs = append(allIDs, params.TrackIDs...)

	// Jellyfin doesn't allow bulk setting favorites.
	// To not overwhelm the server with requests, set favorite for
	// only 5 items at a time concurrently
	batchSize := 5
	var err error
	batchSetFavorite := func(offs int, wg *sync.WaitGroup) {
		for i := 0; i < batchSize && offs+i < len(allIDs); i++ {
			wg.Add(1)
			go func(idx int) {
				newErr := j.client.SetFavorite(allIDs[idx], favorite)
				if err == nil && newErr != nil {
					err = newErr
				}
				wg.Done()
			}(offs + i)
		}
	}

	numBatches := int(math.Ceil(float64(len(allIDs)) / float64(batchSize)))
	for i := 0; i < numBatches; i++ {
		var wg sync.WaitGroup
		batchSetFavorite(i*batchSize, &wg)
		wg.Wait()
	}

	return err
}

func (j *jellyfinMediaProvider) SetRating(params mediaprovider.RatingFavoriteParameters, rating int) error {
	return errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) GetStreamURL(trackID string, forceRaw bool) (string, error) {
	return j.client.GetStreamURL(trackID)
}

func (j *jellyfinMediaProvider) DownloadTrack(trackID string) (io.Reader, error) {
	return nil, errors.New("unimplemented")
}

func (j *jellyfinMediaProvider) Scrobble(trackID string, submission bool) error {
	return errors.New("unimplemented")
}

func (s *jellyfinMediaProvider) RescanLibrary() error {
	return errors.ErrUnsupported
}

func toTrack(ch *jellyfin.Song) *mediaprovider.Track {
	if ch == nil {
		return nil
	}
	var artistNames, artistIDs []string
	for _, a := range ch.Artists {
		artistIDs = append(artistIDs, a.ID)
		artistNames = append(artistNames, a.Name)
	}

	t := &mediaprovider.Track{
		ID: ch.Id,
		//CoverArtID:  ch.CoverArt,
		ParentID:    ch.AlbumID,
		Name:        ch.Name,
		Duration:    int(ch.RunTimeTicks / 1_000_000),
		TrackNumber: ch.IndexNumber,
		DiscNumber:  ch.DiscNumber,
		//Genre:       ch.Genres,
		ArtistIDs:   artistIDs,
		ArtistNames: artistNames,
		Album:       ch.Album,
		AlbumID:     ch.AlbumID,
		Year:        ch.ProductionYear,
		Rating:      ch.UserData.Rating,
		Favorite:    ch.UserData.IsFavorite,
		PlayCount:   ch.UserData.PlayCount,
	}
	if len(ch.MediaSources) > 0 {
		t.FilePath = ch.MediaSources[0].Path
		t.Size = int64(ch.MediaSources[0].Size)
		t.BitRate = ch.MediaSources[0].Bitrate
	}
	return t
}

func toArtist(a *jellyfin.Artist) *mediaprovider.Artist {
	art := &mediaprovider.Artist{}
	fillArtist(a, art)
	return art
}

func fillArtist(a *jellyfin.Artist, artist *mediaprovider.Artist) {
	artist.AlbumCount = a.AlbumCount
	artist.Favorite = a.UserData.IsFavorite
	artist.ID = a.ID
	artist.Name = a.Name
	artist.CoverArtID = a.ID
}

func toAlbum(a *jellyfin.Album) *mediaprovider.Album {
	album := &mediaprovider.Album{}
	fillAlbum(a, album)
	return album
}

func fillAlbum(a *jellyfin.Album, album *mediaprovider.Album) {
	var artistNames, artistIDs []string
	for _, a := range a.Artists {
		artistIDs = append(artistIDs, a.ID)
		artistNames = append(artistNames, a.Name)
	}

	album.ID = a.ID
	//album.CoverArtID = a.CoverArt
	album.Name = a.Name
	album.Duration = int(a.RunTimeTicks / 1_000_000)
	album.ArtistIDs = artistIDs
	album.ArtistNames = artistNames
	album.Year = a.Year
	//album.TrackCount = a.
	album.Genres = a.Genres
	album.Favorite = a.UserData.IsFavorite
}

func toPlaylist(p *jellyfin.Playlist) *mediaprovider.Playlist {
	pl := &mediaprovider.Playlist{}
	fillPlaylist(p, pl)
	return pl
}

func fillPlaylist(p *jellyfin.Playlist, pl *mediaprovider.Playlist) {
	pl.Name = p.Name
	pl.ID = p.ID
	//CoverArtID = pl.CoverArt
	pl.Description = p.Overview
	//.Owner = pl.Owner
	//Public = pl.Public
	pl.TrackCount = p.SongCount
	pl.Duration = int(p.RunTimeTicks / 1_000_000)
}
