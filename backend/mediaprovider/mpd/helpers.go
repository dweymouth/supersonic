package mpd

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/fhs/gompd/v2/mpd"
)

// ID Encoding Scheme:
// - Tracks: file path (already unique in MPD)
// - Albums: "album::<albumname>::<artist>" (URL-escaped)
// - Artists: "artist::<artistname>" (URL-escaped)
// - Playlists: playlist name

const (
	albumIDPrefix  = "album::"
	artistIDPrefix = "artist::"
)

// encodeAlbumID creates a unique ID for an album from its name and artist.
func encodeAlbumID(albumName, artist string) string {
	return albumIDPrefix + url.PathEscape(albumName) + "::" + url.PathEscape(artist)
}

// decodeAlbumID extracts album name and artist from an encoded album ID.
func decodeAlbumID(id string) (albumName, artist string, ok bool) {
	if !strings.HasPrefix(id, albumIDPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(id, albumIDPrefix)
	parts := strings.SplitN(rest, "::", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	albumName, _ = url.PathUnescape(parts[0])
	artist, _ = url.PathUnescape(parts[1])
	return albumName, artist, true
}

// encodeArtistID creates a unique ID for an artist from their name.
func encodeArtistID(artistName string) string {
	return artistIDPrefix + url.PathEscape(artistName)
}

// decodeArtistID extracts the artist name from an encoded artist ID.
func decodeArtistID(id string) (artistName string, ok bool) {
	if !strings.HasPrefix(id, artistIDPrefix) {
		return "", false
	}
	artistName, _ = url.PathUnescape(strings.TrimPrefix(id, artistIDPrefix))
	return artistName, true
}

// toTrack converts MPD attributes to a mediaprovider.Track.
func toTrack(attrs mpd.Attrs) *mediaprovider.Track {
	if attrs == nil {
		return nil
	}

	file := attrs["file"]
	title := attrs["Title"]
	if title == "" {
		// Use filename without extension as fallback
		title = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	}

	artist := attrs["Artist"]
	albumArtist := attrs["AlbumArtist"]
	if albumArtist == "" {
		albumArtist = artist
	}

	album := attrs["Album"]
	albumID := ""
	if album != "" {
		albumID = encodeAlbumID(album, albumArtist)
	}

	// Parse duration
	var duration time.Duration
	if durationStr := attrs["duration"]; durationStr != "" {
		duration = parseSeconds(durationStr)
	} else if timeStr := attrs["Time"]; timeStr != "" {
		if secs, err := strconv.Atoi(timeStr); err == nil {
			duration = time.Duration(secs) * time.Second
		}
	}

	// Parse track and disc numbers
	trackNum, _ := strconv.Atoi(attrs["Track"])
	discNum, _ := strconv.Atoi(attrs["Disc"])

	// Parse year from Date tag
	year := 0
	if dateStr := attrs["Date"]; dateStr != "" {
		// Try to parse just the year (first 4 characters)
		if len(dateStr) >= 4 {
			year, _ = strconv.Atoi(dateStr[:4])
		}
	}

	// Handle multiple artists (semicolon or slash separated)
	var artistNames, artistIDs []string
	if artist != "" {
		artistNames = splitArtists(artist)
		for _, a := range artistNames {
			artistIDs = append(artistIDs, encodeArtistID(a))
		}
	}

	var albumArtistNames, albumArtistIDs []string
	if albumArtist != "" {
		albumArtistNames = splitArtists(albumArtist)
		for _, a := range albumArtistNames {
			albumArtistIDs = append(albumArtistIDs, encodeArtistID(a))
		}
	}

	// Genres
	var genres []string
	if genre := attrs["Genre"]; genre != "" {
		genres = splitGenres(genre)
	}

	// Composers (MPD tag: Composer)
	var composerNames []string
	if composer := attrs["Composer"]; composer != "" {
		composerNames = splitArtists(composer)
	}

	// Comment
	comment := attrs["Comment"]

	// File extension and content type
	ext := strings.TrimPrefix(filepath.Ext(file), ".")
	contentType := extensionToContentType(ext)

	// For cover art, use the album ID if available
	coverArtID := albumID
	if coverArtID == "" {
		coverArtID = file
	}

	return &mediaprovider.Track{
		ID:               file,
		CoverArtID:       coverArtID,
		ParentID:         albumID,
		Title:            title,
		Duration:         duration,
		TrackNumber:      trackNum,
		DiscNumber:       discNum,
		Genres:           genres,
		ArtistIDs:        artistIDs,
		ArtistNames:      artistNames,
		AlbumArtistIDs:   albumArtistIDs,
		AlbumArtistNames: albumArtistNames,
		ComposerNames:    composerNames,
		Comment:          comment,
		Album:            album,
		AlbumID:          albumID,
		Year:             year,
		FilePath:         file,
		Extension:        ext,
		ContentType:      contentType,
	}
}

// toAlbum creates an Album from track attributes (typically the first track of the album).
func toAlbum(albumName, artistName string, trackCount int, duration time.Duration) *mediaprovider.Album {
	if albumName == "" {
		return nil
	}

	id := encodeAlbumID(albumName, artistName)
	var artistIDs, artistNames []string
	if artistName != "" {
		artistNames = splitArtists(artistName)
		for _, a := range artistNames {
			artistIDs = append(artistIDs, encodeArtistID(a))
		}
	}

	return &mediaprovider.Album{
		ID:          id,
		CoverArtID:  id,
		Name:        albumName,
		ArtistIDs:   artistIDs,
		ArtistNames: artistNames,
		TrackCount:  trackCount,
		Duration:    duration,
	}
}

// toArtist creates an Artist from a name and album count.
// coverArtID can be provided to use an album cover as the artist image fallback.
func toArtist(artistName string, albumCount int, coverArtID string) *mediaprovider.Artist {
	if artistName == "" {
		return nil
	}

	id := encodeArtistID(artistName)
	return &mediaprovider.Artist{
		ID:         id,
		CoverArtID: coverArtID, // Use first album's cover as artist image
		Name:       artistName,
		AlbumCount: albumCount,
	}
}

// toPlaylist converts MPD playlist attributes to a mediaprovider.Playlist.
func toPlaylist(name string, trackCount int) *mediaprovider.Playlist {
	return &mediaprovider.Playlist{
		ID:         name,
		Name:       name,
		TrackCount: trackCount,
	}
}

// splitArtists splits an artist string by common separators.
func splitArtists(artist string) []string {
	// Try common separators: semicolon, slash, ampersand
	if strings.Contains(artist, ";") {
		parts := strings.Split(artist, ";")
		return trimStrings(parts)
	}
	if strings.Contains(artist, " / ") {
		parts := strings.Split(artist, " / ")
		return trimStrings(parts)
	}
	if strings.Contains(artist, " & ") {
		parts := strings.Split(artist, " & ")
		return trimStrings(parts)
	}
	return []string{artist}
}

// splitGenres splits a genre string by common separators.
func splitGenres(genre string) []string {
	if strings.Contains(genre, ";") {
		parts := strings.Split(genre, ";")
		return trimStrings(parts)
	}
	if strings.Contains(genre, ",") {
		parts := strings.Split(genre, ",")
		return trimStrings(parts)
	}
	if strings.Contains(genre, "/") {
		parts := strings.Split(genre, "/")
		return trimStrings(parts)
	}
	return []string{genre}
}

// trimStrings trims whitespace from each string and removes empty strings.
func trimStrings(ss []string) []string {
	var result []string
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// albumFromTracks creates an Album with aggregated information from tracks.
func albumFromTracks(albumName, artistName string, tracks []mpd.Attrs) *mediaprovider.Album {
	var totalDuration time.Duration
	var year int
	genreSet := make(map[string]struct{})

	for _, track := range tracks {
		// Sum duration
		if durationStr := track["duration"]; durationStr != "" {
			totalDuration += parseSeconds(durationStr)
		} else if timeStr := track["Time"]; timeStr != "" {
			if secs, err := strconv.Atoi(timeStr); err == nil {
				totalDuration += time.Duration(secs) * time.Second
			}
		}

		// Get year from first track that has it
		if year == 0 {
			if dateStr := track["Date"]; dateStr != "" && len(dateStr) >= 4 {
				year, _ = strconv.Atoi(dateStr[:4])
			}
		}

		// Collect genres from all tracks
		if genre := track["Genre"]; genre != "" {
			for _, g := range splitGenres(genre) {
				genreSet[g] = struct{}{}
			}
		}
	}

	album := toAlbum(albumName, artistName, len(tracks), totalDuration)
	if album != nil {
		if year > 0 {
			album.Date.Year = &year
		}
		// Convert genre set to slice
		if len(genreSet) > 0 {
			genres := make([]string, 0, len(genreSet))
			for g := range genreSet {
				genres = append(genres, g)
			}
			album.Genres = genres
		}
	}

	return album
}

// albumTracksToTracks converts a slice of MPD attrs to Track objects.
func albumTracksToTracks(attrs []mpd.Attrs) []*mediaprovider.Track {
	tracks := make([]*mediaprovider.Track, 0, len(attrs))
	for _, a := range attrs {
		if track := toTrack(a); track != nil {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

// extensionToContentType maps file extensions to MIME content types.
func extensionToContentType(ext string) string {
	switch strings.ToLower(ext) {
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "ogg", "oga":
		return "audio/ogg"
	case "opus":
		return "audio/opus"
	case "m4a", "mp4", "aac":
		return "audio/mp4"
	case "wav":
		return "audio/wav"
	case "wma":
		return "audio/x-ms-wma"
	case "aiff", "aif":
		return "audio/aiff"
	case "ape":
		return "audio/ape"
	case "wv":
		return "audio/wavpack"
	case "mpc":
		return "audio/musepack"
	case "dsf", "dff":
		return "audio/dsd"
	default:
		return "audio/" + strings.ToLower(ext)
	}
}
