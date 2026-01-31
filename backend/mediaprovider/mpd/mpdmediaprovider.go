package mpd

import (
	"fmt"
	"image"
	"io"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/mediaprovider/helpers"
	"github.com/fhs/gompd/v2/mpd"
)

const (
	playlistCacheValidDurationSeconds = 60
	genresCacheValidDurationSeconds   = 120
)

// mpdMediaProvider implements mediaprovider.MediaProvider for MPD.
type mpdMediaProvider struct {
	server            *MPDServer
	prefetchCoverCB   func(coverArtID string)
	artistInfoFetcher *artistInfoFetcher

	genresCached   []*mediaprovider.Genre
	genresCachedAt int64

	playlistsCached   []*mediaprovider.Playlist
	playlistsCachedAt int64
}

// Ensure mpdMediaProvider implements MediaProvider
var _ mediaprovider.MediaProvider = (*mpdMediaProvider)(nil)

// Ensure mpdMediaProvider implements JukeboxProvider
var _ mediaprovider.JukeboxProvider = (*mpdMediaProvider)(nil)

// Ensure mpdMediaProvider implements JukeboxOnlyServer
var _ mediaprovider.JukeboxOnlyServer = (*mpdMediaProvider)(nil)

// Ensure mpdMediaProvider implements SupportsRating
var _ mediaprovider.SupportsRating = (*mpdMediaProvider)(nil)

// Ensure mpdMediaProvider implements CacheManager
var _ mediaprovider.CacheManager = (*mpdMediaProvider)(nil)

// IsJukeboxOnly returns true because MPD only supports jukebox mode.
func (m *mpdMediaProvider) IsJukeboxOnly() bool {
	return true
}

// ClearCaches clears the artist info cache (Deezer/Wikipedia data).
func (m *mpdMediaProvider) ClearCaches() {
	if m.artistInfoFetcher != nil {
		m.artistInfoFetcher.clearCache()
	}
}

func (m *mpdMediaProvider) SetPrefetchCoverCallback(cb func(coverArtID string)) {
	m.prefetchCoverCB = cb
}

func (m *mpdMediaProvider) GetLibraries() ([]mediaprovider.Library, error) {
	// MPD doesn't have multiple libraries concept
	return []mediaprovider.Library{
		{ID: "", Name: "Music"},
	}, nil
}

func (m *mpdMediaProvider) SetLibrary(id string) error {
	// MPD doesn't support multiple libraries
	return nil
}

func (m *mpdMediaProvider) GetTrack(trackID string) (*mediaprovider.Track, error) {
	var track *mediaprovider.Track
	err := m.server.withConn(func(conn *mpd.Client) error {
		// trackID is the file path in MPD
		attrs, err := conn.ListAllInfo(trackID)
		if err != nil {
			return err
		}
		if len(attrs) == 0 {
			return fmt.Errorf("track not found: %s", trackID)
		}
		track = toTrack(attrs[0])

		// Check if this track is currently playing and get audio info
		currentSong, err := conn.CurrentSong()
		if err == nil && currentSong["file"] == trackID {
			// This track is currently playing - get audio details from status
			status, err := conn.Status()
			if err == nil {
				// Parse bitrate
				if bitrate := status["bitrate"]; bitrate != "" {
					track.BitRate, _ = strconv.Atoi(bitrate)
				}
				// Parse audio format: "samplerate:bits:channels"
				if audio := status["audio"]; audio != "" {
					parts := strings.Split(audio, ":")
					if len(parts) >= 3 {
						track.SampleRate, _ = strconv.Atoi(parts[0])
						track.BitDepth, _ = strconv.Atoi(parts[1])
						track.Channels, _ = strconv.Atoi(parts[2])
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Enrich with sticker data (favorite, rating, play count, last played)
	stickers, _ := m.getTrackStickers(trackID)
	track.Favorite = stickers.Favorite
	track.Rating = stickers.Rating
	track.PlayCount = stickers.PlayCount
	track.LastPlayed = stickers.LastPlayed

	return track, nil
}

func (m *mpdMediaProvider) GetAlbum(albumID string) (*mediaprovider.AlbumWithTracks, error) {
	albumName, artistName, ok := decodeAlbumID(albumID)
	if !ok {
		return nil, fmt.Errorf("invalid album ID: %s", albumID)
	}

	var result *mediaprovider.AlbumWithTracks
	err := m.server.withConn(func(conn *mpd.Client) error {
		// Find all tracks for this album
		var attrs []mpd.Attrs
		var err error
		if artistName != "" {
			attrs, err = conn.Find("album", albumName, "albumartist", artistName)
		} else {
			attrs, err = conn.Find("album", albumName)
		}
		if err != nil {
			return err
		}
		if len(attrs) == 0 {
			return fmt.Errorf("album not found: %s", albumName)
		}

		// Sort tracks by disc and track number
		sort.Slice(attrs, func(i, j int) bool {
			di, ti := parseInt(attrs[i]["Disc"]), parseInt(attrs[i]["Track"])
			dj, tj := parseInt(attrs[j]["Disc"]), parseInt(attrs[j]["Track"])
			if di != dj {
				return di < dj
			}
			return ti < tj
		})

		result = &mediaprovider.AlbumWithTracks{
			Tracks: albumTracksToTracks(attrs),
		}
		album := albumFromTracks(albumName, artistName, attrs)
		if album != nil {
			result.Album = *album
		}

		// Try to get genres from tracks
		genreMap := make(map[string]bool)
		for _, a := range attrs {
			if genre := a["Genre"]; genre != "" {
				for _, g := range splitGenres(genre) {
					genreMap[g] = true
				}
			}
		}
		for g := range genreMap {
			result.Album.Genres = append(result.Album.Genres, g)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Enrich tracks with sticker data (favorite, rating, play count, last played)
	hasAlbumFavorite := false
	for _, track := range result.Tracks {
		stickers, _ := m.getTrackStickers(track.ID)
		track.Favorite = stickers.Favorite // Track-level favorite
		track.Rating = stickers.Rating
		track.PlayCount = stickers.PlayCount
		track.LastPlayed = stickers.LastPlayed
		if stickers.AlbumFavorite {
			hasAlbumFavorite = true
		}
	}
	// Album is considered favorite if it has the album_favorite sticker
	result.Album.Favorite = hasAlbumFavorite

	return result, nil
}

func (m *mpdMediaProvider) GetAlbumInfo(albumID string) (*mediaprovider.AlbumInfo, error) {
	// MPD doesn't have album info (no Last.fm integration)
	return &mediaprovider.AlbumInfo{}, nil
}

func (m *mpdMediaProvider) GetArtist(artistID string) (*mediaprovider.ArtistWithAlbums, error) {
	artistName, ok := decodeArtistID(artistID)
	if !ok {
		return nil, fmt.Errorf("invalid artist ID: %s", artistID)
	}

	var result *mediaprovider.ArtistWithAlbums
	err := m.server.withConn(func(conn *mpd.Client) error {
		// Get all albums for this artist
		albums, err := conn.List("album", "albumartist", artistName)
		if err != nil {
			return err
		}

		// Use first album's cover as artist image
		var coverArtID string
		if len(albums) > 0 {
			coverArtID = encodeAlbumID(albums[0], artistName)
		}

		result = &mediaprovider.ArtistWithAlbums{
			Artist: mediaprovider.Artist{
				ID:         artistID,
				CoverArtID: coverArtID,
				Name:       artistName,
				AlbumCount: len(albums),
			},
		}

		// Get details for each album
		for _, albumName := range albums {
			// Get tracks for this album to compute duration
			attrs, err := conn.Find("album", albumName, "albumartist", artistName)
			if err != nil {
				continue
			}
			album := albumFromTracks(albumName, artistName, attrs)
			if album != nil {
				result.Albums = append(result.Albums, album)
			}
		}

		return nil
	})
	return result, err
}

func (m *mpdMediaProvider) GetArtistTracks(artistID string) ([]*mediaprovider.Track, error) {
	return helpers.GetArtistTracks(m, artistID)
}

func (m *mpdMediaProvider) GetArtistInfo(artistID string) (*mediaprovider.ArtistInfo, error) {
	// Decode artist name from ID
	artistName, ok := decodeArtistID(artistID)
	if !ok {
		return &mediaprovider.ArtistInfo{}, nil
	}

	// Fetch artist info from TheAudioDB
	info, err := m.artistInfoFetcher.fetchArtistInfo(artistName)
	if err != nil {
		// Return empty info instead of failing
		return &mediaprovider.ArtistInfo{}, nil
	}
	return info, nil
}

func (m *mpdMediaProvider) GetPlaylist(playlistID string) (*mediaprovider.PlaylistWithTracks, error) {
	var result *mediaprovider.PlaylistWithTracks
	err := m.server.withConn(func(conn *mpd.Client) error {
		attrs, err := conn.PlaylistContents(playlistID)
		if err != nil {
			return err
		}

		result = &mediaprovider.PlaylistWithTracks{
			Playlist: mediaprovider.Playlist{
				ID:         playlistID,
				Name:       playlistID,
				TrackCount: len(attrs),
			},
			Tracks: albumTracksToTracks(attrs),
		}

		// Calculate total duration
		for _, track := range result.Tracks {
			result.Playlist.Duration += track.Duration
		}

		return nil
	})
	return result, err
}

func (m *mpdMediaProvider) GetCoverArt(coverArtID string, size int) (image.Image, error) {
	var img image.Image
	err := m.server.withConn(func(conn *mpd.Client) error {
		// coverArtID could be an album ID or a file path
		var filePath string
		if albumName, artistName, ok := decodeAlbumID(coverArtID); ok {
			// Find a track from this album to get cover art
			var attrs []mpd.Attrs
			var err error
			if artistName != "" {
				attrs, err = conn.Find("album", albumName, "albumartist", artistName)
			} else {
				attrs, err = conn.Find("album", albumName)
			}
			if err != nil || len(attrs) == 0 {
				return fmt.Errorf("no tracks found for album: %s", albumName)
			}
			filePath = attrs[0]["file"]
		} else {
			filePath = coverArtID
		}

		// Try readpicture first (embedded art)
		data, err := conn.ReadPicture(filePath)
		if err != nil || len(data) == 0 {
			// Fall back to albumart (directory art)
			data, err = conn.AlbumArt(filePath)
			if err != nil || len(data) == 0 {
				return fmt.Errorf("no cover art found for: %s", filePath)
			}
		}

		// Decode the image
		img, _, err = image.Decode(strings.NewReader(string(data)))
		return err
	})
	return img, err
}

func (m *mpdMediaProvider) AlbumSortOrders() []string {
	return []string{
		mediaprovider.AlbumSortTitleAZ,
		mediaprovider.AlbumSortArtistAZ,
		mediaprovider.AlbumSortYearAscending,
		mediaprovider.AlbumSortYearDescending,
		mediaprovider.AlbumSortRecentlyPlayed,
		mediaprovider.AlbumSortFrequentlyPlayed,
		mediaprovider.AlbumSortRandom,
	}
}

func (m *mpdMediaProvider) IterateAlbums(sortOrder string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return newAlbumIterator(m, sortOrder, filter)
}

func (m *mpdMediaProvider) IterateTracks(searchQuery string) mediaprovider.TrackIterator {
	return newTrackIterator(m, searchQuery)
}

func (m *mpdMediaProvider) SearchAlbums(searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return newSearchAlbumIterator(m, searchQuery, filter)
}

func (m *mpdMediaProvider) SearchAll(searchQuery string, maxResults int) ([]*mediaprovider.SearchResult, error) {
	var results []*mediaprovider.SearchResult

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Search tracks
		trackAttrs, err := conn.Search("any", searchQuery)
		if err != nil {
			return err
		}

		// Track unique albums and artists found
		albumMap := make(map[string]bool)
		artistMap := make(map[string]bool)

		// Process tracks
		for _, attrs := range trackAttrs {
			track := toTrack(attrs)
			if track == nil {
				continue
			}

			// Add track result
			if len(results) < maxResults {
				results = append(results, &mediaprovider.SearchResult{
					Name:       track.Title,
					ID:         track.ID,
					CoverID:    track.CoverArtID,
					Type:       mediaprovider.ContentTypeTrack,
					Size:       int(track.Duration.Seconds()),
					ArtistName: strings.Join(track.ArtistNames, ", "),
					Item:       track,
				})
			}

			// Track albums
			if track.AlbumID != "" && !albumMap[track.AlbumID] {
				albumMap[track.AlbumID] = true
			}

			// Track artists
			for _, artistID := range track.ArtistIDs {
				if !artistMap[artistID] {
					artistMap[artistID] = true
				}
			}
		}

		// Add album results
		for albumID := range albumMap {
			if len(results) >= maxResults*3 {
				break
			}
			albumName, artistName, _ := decodeAlbumID(albumID)
			results = append(results, &mediaprovider.SearchResult{
				Name:       albumName,
				ID:         albumID,
				CoverID:    albumID,
				Type:       mediaprovider.ContentTypeAlbum,
				ArtistName: artistName,
			})
		}

		// Add artist results
		for artistID := range artistMap {
			if len(results) >= maxResults*4 {
				break
			}
			artistName, _ := decodeArtistID(artistID)
			results = append(results, &mediaprovider.SearchResult{
				Name:    artistName,
				ID:      artistID,
				CoverID: "",
				Type:    mediaprovider.ContentTypeArtist,
			})
		}

		return nil
	})

	// Rank results by relevance
	queryTerms := strings.Fields(strings.ToLower(searchQuery))
	helpers.RankSearchResults(results, strings.ToLower(searchQuery), queryTerms)

	return results, err
}

func (m *mpdMediaProvider) GetRandomTracks(genre string, count int) ([]*mediaprovider.Track, error) {
	var tracks []*mediaprovider.Track
	err := m.server.withConn(func(conn *mpd.Client) error {
		var attrs []mpd.Attrs
		var err error
		if genre != "" {
			attrs, err = conn.Find("genre", genre)
		} else {
			attrs, err = conn.ListAllInfo("/")
		}
		if err != nil {
			return err
		}

		// Filter to actual files (not directories)
		var fileAttrs []mpd.Attrs
		for _, a := range attrs {
			if a["file"] != "" {
				fileAttrs = append(fileAttrs, a)
			}
		}

		// Shuffle and take count
		shuffled := make([]mpd.Attrs, len(fileAttrs))
		copy(shuffled, fileAttrs)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		if len(shuffled) > count {
			shuffled = shuffled[:count]
		}

		tracks = albumTracksToTracks(shuffled)
		return nil
	})
	return tracks, err
}

func (m *mpdMediaProvider) GetSimilarTracks(artistID string, count int) ([]*mediaprovider.Track, error) {
	// MPD doesn't have similarity/recommendation features
	// Fall back to random tracks by the same artist
	artistName, ok := decodeArtistID(artistID)
	if !ok {
		return nil, fmt.Errorf("invalid artist ID: %s", artistID)
	}

	var tracks []*mediaprovider.Track
	err := m.server.withConn(func(conn *mpd.Client) error {
		attrs, err := conn.Find("artist", artistName)
		if err != nil {
			return err
		}

		// Shuffle and take count
		shuffled := make([]mpd.Attrs, len(attrs))
		copy(shuffled, attrs)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		if len(shuffled) > count {
			shuffled = shuffled[:count]
		}

		tracks = albumTracksToTracks(shuffled)
		return nil
	})
	return tracks, err
}

func (m *mpdMediaProvider) GetSongRadio(trackID string, count int) ([]*mediaprovider.Track, error) {
	// MPD doesn't have song radio feature
	// Fall back to similar tracks by artist
	track, err := m.GetTrack(trackID)
	if err != nil {
		return nil, err
	}

	if len(track.ArtistIDs) > 0 {
		return m.GetSimilarTracks(track.ArtistIDs[0], count)
	}

	// Fall back to random tracks of same genre
	genre := ""
	if len(track.Genres) > 0 {
		genre = track.Genres[0]
	}
	return m.GetRandomTracks(genre, count)
}

func (m *mpdMediaProvider) ArtistSortOrders() []string {
	return []string{
		mediaprovider.ArtistSortNameAZ,
		mediaprovider.ArtistSortAlbumCount,
	}
}

func (m *mpdMediaProvider) IterateArtists(sortOrder string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	return newArtistIterator(m, sortOrder, filter)
}

func (m *mpdMediaProvider) SearchArtists(searchQuery string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	return newSearchArtistIterator(m, searchQuery, filter)
}

func (m *mpdMediaProvider) GetGenres() ([]*mediaprovider.Genre, error) {
	if m.genresCached != nil && time.Now().Unix()-m.genresCachedAt < genresCacheValidDurationSeconds {
		return m.genresCached, nil
	}

	// Map to aggregate counts for split genres
	genreMap := make(map[string]*mediaprovider.Genre)

	err := m.server.withConn(func(conn *mpd.Client) error {
		genreNames, err := conn.List("genre")
		if err != nil {
			return err
		}

		for _, rawGenre := range genreNames {
			if rawGenre == "" {
				continue
			}

			// Get tracks for this raw genre tag
			tracks, err := conn.Find("genre", rawGenre)
			if err != nil {
				continue
			}

			// Split the genre string (e.g., "Rock, Pop" -> ["Rock", "Pop"])
			splitGenreNames := splitGenres(rawGenre)

			for _, genreName := range splitGenreNames {
				if genreName == "" {
					continue
				}

				// Get or create genre entry
				genre, exists := genreMap[strings.ToLower(genreName)]
				if !exists {
					genre = &mediaprovider.Genre{
						Name:       genreName,
						AlbumCount: 0,
						TrackCount: 0,
					}
					genreMap[strings.ToLower(genreName)] = genre
				}

				// Count unique albums and tracks for this split genre
				albumSet := make(map[string]bool)
				for _, t := range tracks {
					if album := t["Album"]; album != "" {
						artist := t["AlbumArtist"]
						if artist == "" {
							artist = t["Artist"]
						}
						albumSet[encodeAlbumID(album, artist)] = true
					}
				}
				genre.TrackCount += len(tracks)
				genre.AlbumCount += len(albumSet)
			}
		}
		return nil
	})

	// Convert map to slice
	var genres []*mediaprovider.Genre
	for _, g := range genreMap {
		genres = append(genres, g)
	}

	if err == nil {
		m.genresCached = genres
		m.genresCachedAt = time.Now().Unix()
	}

	return genres, err
}

func (m *mpdMediaProvider) GetFavorites() (mediaprovider.Favorites, error) {
	// Get favorited tracks using stickers
	tracks, err := m.getFavoriteTracks()
	if err != nil {
		return mediaprovider.Favorites{}, err
	}

	// Get albums that have been favorited
	albums, err := m.getFavoriteAlbums()
	if err != nil {
		return mediaprovider.Favorites{}, err
	}

	// Get artists that have been favorited
	artists, err := m.getFavoriteArtists()
	if err != nil {
		return mediaprovider.Favorites{}, err
	}

	return mediaprovider.Favorites{
		Tracks:  tracks,
		Albums:  albums,
		Artists: artists,
	}, nil
}

func (m *mpdMediaProvider) GetStreamURL(trackID string, transcodeSettings *mediaprovider.TranscodeSettings, forceRaw bool) (string, error) {
	// MPD doesn't provide streaming URLs - playback is through jukebox
	return "", ErrNotSupported
}

func (m *mpdMediaProvider) GetTopTracks(artist mediaprovider.Artist, count int) ([]*mediaprovider.Track, error) {
	// Get all tracks for this artist
	tracks, err := m.GetArtistTracks(artist.ID)
	if err != nil {
		return helpers.GetTopTracksFallback(m, artist.ID, count)
	}

	// Enrich with play count stickers
	for _, track := range tracks {
		stickers, _ := m.getTrackStickers(track.ID)
		track.PlayCount = stickers.PlayCount
		track.LastPlayed = stickers.LastPlayed
		track.Favorite = stickers.Favorite
		track.Rating = stickers.Rating
	}

	// Sort by play count (descending)
	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].PlayCount > tracks[j].PlayCount
	})

	// Return top tracks (only those with play counts > 0, or fallback to all)
	var topTracks []*mediaprovider.Track
	for _, track := range tracks {
		if track.PlayCount > 0 {
			topTracks = append(topTracks, track)
		}
		if len(topTracks) >= count {
			break
		}
	}

	// If no tracks have play counts, return first N tracks
	if len(topTracks) == 0 && len(tracks) > 0 {
		if len(tracks) > count {
			return tracks[:count], nil
		}
		return tracks, nil
	}

	return topTracks, nil
}

func (m *mpdMediaProvider) SetFavorite(params mediaprovider.RatingFavoriteParameters, favorite bool) error {
	// Set favorite for tracks using stickers
	for _, trackID := range params.TrackIDs {
		if err := m.setTrackFavorite(trackID, favorite); err != nil {
			return err
		}
	}

	// Set favorite for all tracks in albums
	for _, albumID := range params.AlbumIDs {
		if err := m.setAlbumFavorite(albumID, favorite); err != nil {
			return err
		}
	}

	// Set favorite for all tracks by artists
	for _, artistID := range params.ArtistIDs {
		if err := m.setArtistFavorite(artistID, favorite); err != nil {
			return err
		}
	}

	return nil
}

// SetRating implements SupportsRating interface using MPD stickers.
func (m *mpdMediaProvider) SetRating(params mediaprovider.RatingFavoriteParameters, rating int) error {
	// Set rating for tracks using stickers
	for _, trackID := range params.TrackIDs {
		if err := m.setTrackRating(trackID, rating); err != nil {
			return err
		}
	}

	// Set rating for all tracks in albums
	for _, albumID := range params.AlbumIDs {
		if err := m.setAlbumRating(albumID, rating); err != nil {
			return err
		}
	}

	return nil
}

func (m *mpdMediaProvider) GetPlaylists() ([]*mediaprovider.Playlist, error) {
	if m.playlistsCached != nil && time.Now().Unix()-m.playlistsCachedAt < playlistCacheValidDurationSeconds {
		return m.playlistsCached, nil
	}

	var playlists []*mediaprovider.Playlist
	err := m.server.withConn(func(conn *mpd.Client) error {
		attrs, err := conn.ListPlaylists()
		if err != nil {
			return err
		}

		for _, a := range attrs {
			name := a["playlist"]
			if name == "" {
				continue
			}
			// Get track count
			contents, _ := conn.PlaylistContents(name)
			playlists = append(playlists, toPlaylist(name, len(contents)))
		}
		return nil
	})

	if err == nil {
		m.playlistsCached = playlists
		m.playlistsCachedAt = time.Now().Unix()
	}

	return playlists, err
}

func (m *mpdMediaProvider) CreatePlaylistWithTracks(name string, trackIDs []string) error {
	m.playlistsCached = nil
	return m.server.withConn(func(conn *mpd.Client) error {
		// Create empty playlist
		if err := conn.PlaylistClear(name); err != nil {
			// Playlist might not exist, ignore error
		}
		// Add tracks
		for _, trackID := range trackIDs {
			if err := conn.PlaylistAdd(name, trackID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *mpdMediaProvider) CanMakePublicPlaylist() bool {
	// MPD playlists are local, not public/private
	return false
}

func (m *mpdMediaProvider) CreatePlaylist(name, description string, public bool) error {
	m.playlistsCached = nil
	// MPD playlists are just names, no description or public flag
	return m.server.withConn(func(conn *mpd.Client) error {
		// Create empty playlist by saving current (empty) queue
		return conn.PlaylistSave(name)
	})
}

func (m *mpdMediaProvider) EditPlaylist(id, name, description string, public bool) error {
	// MPD can only rename playlists
	if id != name {
		m.playlistsCached = nil
		return m.server.withConn(func(conn *mpd.Client) error {
			return conn.PlaylistRename(id, name)
		})
	}
	return nil
}

func (m *mpdMediaProvider) AddPlaylistTracks(id string, trackIDsToAdd []string) error {
	m.playlistsCached = nil
	return m.server.withConn(func(conn *mpd.Client) error {
		for _, trackID := range trackIDsToAdd {
			if err := conn.PlaylistAdd(id, trackID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *mpdMediaProvider) RemovePlaylistTracks(id string, trackIdxsToRemove []int) error {
	m.playlistsCached = nil
	return m.server.withConn(func(conn *mpd.Client) error {
		// Remove in reverse order to maintain correct indices
		sort.Sort(sort.Reverse(sort.IntSlice(trackIdxsToRemove)))
		for _, idx := range trackIdxsToRemove {
			if err := conn.PlaylistDelete(id, idx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *mpdMediaProvider) ReplacePlaylistTracks(id string, trackIDs []string) error {
	m.playlistsCached = nil
	return m.server.withConn(func(conn *mpd.Client) error {
		// Clear and re-add
		if err := conn.PlaylistClear(id); err != nil {
			return err
		}
		for _, trackID := range trackIDs {
			if err := conn.PlaylistAdd(id, trackID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *mpdMediaProvider) DeletePlaylist(id string) error {
	m.playlistsCached = nil
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.PlaylistRemove(id)
	})
}

func (m *mpdMediaProvider) ClientDecidesScrobble() bool {
	// MPD doesn't do scrobbling - we won't send scrobbles
	return true
}

func (m *mpdMediaProvider) TrackBeganPlayback(trackID string) error {
	// No-op for MPD - we track plays at the end
	return nil
}

func (m *mpdMediaProvider) TrackEndedPlayback(trackID string, positionSecs int, submission bool) error {
	// If this is a submission (full play), increment the play count
	if submission {
		return m.incrementPlayCount(trackID)
	}
	return nil
}

func (m *mpdMediaProvider) DownloadTrack(trackID string) (io.Reader, error) {
	// MPD doesn't support downloading tracks
	return nil, ErrNotSupported
}

func (m *mpdMediaProvider) RescanLibrary() error {
	return m.server.withConn(func(conn *mpd.Client) error {
		_, err := conn.Update("")
		return err
	})
}

// Helper function to get all albums from MPD
func (m *mpdMediaProvider) getAllAlbums() ([]*mediaprovider.Album, error) {
	var albums []*mediaprovider.Album
	err := m.server.withConn(func(conn *mpd.Client) error {
		// Get all album names first
		albumNames, err := conn.List("album")
		if err != nil {
			return err
		}

		// Build a map to track unique album+artist combinations
		albumMap := make(map[string]*mediaprovider.Album)

		for _, albumName := range albumNames {
			if albumName == "" {
				continue
			}

			// Find tracks for this album
			attrs, err := conn.Find("album", albumName)
			if err != nil || len(attrs) == 0 {
				continue
			}

			// Group tracks by album artist (same album name can have different artists)
			artistTracks := make(map[string][]mpd.Attrs)
			for _, a := range attrs {
				artist := a["AlbumArtist"]
				if artist == "" {
					artist = a["Artist"]
				}
				artistTracks[artist] = append(artistTracks[artist], a)
			}

			// Create an album for each artist
			for artist, tracks := range artistTracks {
				albumID := encodeAlbumID(albumName, artist)
				if albumMap[albumID] != nil {
					continue // Already processed
				}

				album := albumFromTracks(albumName, artist, tracks)
				if album != nil {
					albumMap[albumID] = album
				}
			}
		}

		// Convert map to slice
		for _, album := range albumMap {
			albums = append(albums, album)
		}

		return nil
	})
	return albums, err
}

// Helper function to get all artists from MPD
func (m *mpdMediaProvider) getAllArtists() ([]*mediaprovider.Artist, error) {
	var artists []*mediaprovider.Artist
	err := m.server.withConn(func(conn *mpd.Client) error {
		artistNames, err := conn.List("albumartist")
		if err != nil {
			return err
		}

		for _, name := range artistNames {
			if name == "" {
				continue
			}
			// Get album list for this artist
			albums, _ := conn.List("album", "albumartist", name)

			// Use the first album's cover as the artist image
			var coverArtID string
			if len(albums) > 0 {
				coverArtID = encodeAlbumID(albums[0], name)
			}

			artists = append(artists, toArtist(name, len(albums), coverArtID))
		}
		return nil
	})
	return artists, err
}

// albumPlayStats holds aggregated play statistics for an album.
type albumPlayStats struct {
	playCount  int
	lastPlayed time.Time
}

// getAlbumPlayStats retrieves play statistics for a list of albums.
// Returns a map of album ID to play stats (total play count and most recent play time).
func (m *mpdMediaProvider) getAlbumPlayStats(albums []*mediaprovider.Album) map[string]albumPlayStats {
	stats := make(map[string]albumPlayStats)

	// Initialize all albums with zero stats
	for _, album := range albums {
		stats[album.ID] = albumPlayStats{}
	}

	// Query all tracks with play count stickers
	m.server.withConn(func(conn *mpd.Client) error {
		// Find all files with playcount sticker
		// Note: MPD expects empty string "" for root, not "/"
		uris, stickers, err := conn.StickerFind("", stickerPlayCount)
		if err != nil {
			return nil // Ignore errors, return zero stats
		}

		// Build a map of file -> play count
		playCountMap := make(map[string]int)
		for i, uri := range uris {
			if i < len(stickers) {
				if count, err := strconv.Atoi(stickers[i].Value); err == nil {
					playCountMap[uri] = count
				}
			}
		}

		// Find all files with lastplayed sticker
		uris, stickers, err = conn.StickerFind("", stickerLastPlayed)
		if err == nil {
			// Build a map of file -> last played
			lastPlayedMap := make(map[string]time.Time)
			for i, uri := range uris {
				if i < len(stickers) {
					if ts, err := strconv.ParseInt(stickers[i].Value, 10, 64); err == nil {
						lastPlayedMap[uri] = time.Unix(ts, 0)
					}
				}
			}

			// For each album, aggregate stats from its tracks
			for _, album := range albums {
				albumName, artistName, ok := decodeAlbumID(album.ID)
				if !ok {
					continue
				}

				// Find tracks for this album
				var attrs []mpd.Attrs
				if artistName != "" {
					attrs, _ = conn.Find("album", albumName, "albumartist", artistName)
				} else {
					attrs, _ = conn.Find("album", albumName)
				}

				var totalPlayCount int
				var latestPlay time.Time

				for _, a := range attrs {
					file := a["file"]
					if file == "" {
						continue
					}

					// Add play count
					if count, ok := playCountMap[file]; ok {
						totalPlayCount += count
					}

					// Track latest play time
					if lp, ok := lastPlayedMap[file]; ok && lp.After(latestPlay) {
						latestPlay = lp
					}
				}

				stats[album.ID] = albumPlayStats{
					playCount:  totalPlayCount,
					lastPlayed: latestPlay,
				}
			}
		}

		return nil
	})

	return stats
}

