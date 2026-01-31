package mpd

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/fhs/gompd/v2/mpd"
)

// Sticker names used for track metadata
const (
	stickerFavorite       = "favorite"        // Track-level favorite
	stickerAlbumFavorite  = "album_favorite"  // Album-level favorite (set on tracks to mark album as favorite)
	stickerArtistFavorite = "artist_favorite" // Artist-level favorite (set on tracks to mark artist as favorite)
	stickerRating         = "rating"
	stickerPlayCount      = "playcount"
	stickerLastPlayed     = "lastplayed"
)

// setTrackFavorite sets or removes the favorite sticker for a track.
func (m *mpdMediaProvider) setTrackFavorite(trackID string, favorite bool) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		if favorite {
			return conn.StickerSet(trackID, stickerFavorite, "1")
		}
		// Delete the sticker if unfavoriting
		err := conn.StickerDelete(trackID, stickerFavorite)
		// Ignore "no such sticker" errors when deleting
		if err != nil && !isNoStickerError(err) {
			return err
		}
		return nil
	})
}

// setTrackRating sets or removes the rating sticker for a track.
func (m *mpdMediaProvider) setTrackRating(trackID string, rating int) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		if rating > 0 && rating <= 5 {
			return conn.StickerSet(trackID, stickerRating, strconv.Itoa(rating))
		}
		// Delete the sticker if rating is 0
		err := conn.StickerDelete(trackID, stickerRating)
		if err != nil && !isNoStickerError(err) {
			return err
		}
		return nil
	})
}

// trackStickers holds all sticker data for a track.
type trackStickers struct {
	Favorite      bool // Track-level favorite
	AlbumFavorite bool // Album-level favorite (track belongs to a favorited album)
	Rating        int
	PlayCount     int
	LastPlayed    time.Time
}

// getTrackStickers retrieves all stickers for a track.
func (m *mpdMediaProvider) getTrackStickers(trackID string) (stickers trackStickers, err error) {
	err = m.server.withConn(func(conn *mpd.Client) error {
		stickerList, listErr := conn.StickerList(trackID)
		if listErr != nil {
			// No stickers is not an error
			if isNoStickerError(listErr) {
				return nil
			}
			return listErr
		}

		for _, s := range stickerList {
			switch s.Name {
			case stickerFavorite:
				stickers.Favorite = s.Value == "1"
			case stickerAlbumFavorite:
				stickers.AlbumFavorite = s.Value == "1"
			case stickerRating:
				stickers.Rating, _ = strconv.Atoi(s.Value)
			case stickerPlayCount:
				stickers.PlayCount, _ = strconv.Atoi(s.Value)
			case stickerLastPlayed:
				// Parse Unix timestamp
				if ts, parseErr := strconv.ParseInt(s.Value, 10, 64); parseErr == nil {
					stickers.LastPlayed = time.Unix(ts, 0)
				}
			}
		}
		return nil
	})
	return
}

// incrementPlayCount increments the play count sticker and updates last played time.
func (m *mpdMediaProvider) incrementPlayCount(trackID string) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		// Get current play count
		currentCount := 0
		stickers, err := conn.StickerList(trackID)
		if err == nil {
			for _, s := range stickers {
				if s.Name == stickerPlayCount {
					currentCount, _ = strconv.Atoi(s.Value)
					break
				}
			}
		}

		// Increment play count
		newCount := currentCount + 1
		if err := conn.StickerSet(trackID, stickerPlayCount, strconv.Itoa(newCount)); err != nil {
			return err
		}

		// Update last played time (Unix timestamp)
		now := time.Now().Unix()
		return conn.StickerSet(trackID, stickerLastPlayed, strconv.FormatInt(now, 10))
	})
}

// getFavoriteTracks returns all tracks that have the favorite sticker set.
func (m *mpdMediaProvider) getFavoriteTracks() ([]*mediaprovider.Track, error) {
	var tracks []*mediaprovider.Track

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Find all files with favorite sticker in the root directory (recursive)
		// Note: MPD expects empty string "" for root, not "/"
		uris, _, err := conn.StickerFind("", stickerFavorite)
		if err != nil {
			if isNoStickerError(err) {
				return nil
			}
			return err
		}

		for _, uri := range uris {
			// Get track info
			attrs, err := conn.ListAllInfo(uri)
			if err != nil {
				log.Printf("Error getting track info for %s: %v", uri, err)
				continue
			}
			if len(attrs) > 0 {
				track := toTrack(attrs[0])
				if track != nil {
					track.Favorite = true
					tracks = append(tracks, track)
				}
			}
		}
		return nil
	})

	return tracks, err
}

// getFavoriteAlbumIDs returns a set of album IDs that have been marked as album favorites.
// This looks for the album_favorite sticker, not the track-level favorite sticker.
func (m *mpdMediaProvider) getFavoriteAlbumIDs() (map[string]struct{}, error) {
	albumIDs := make(map[string]struct{})

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Find all files with album_favorite sticker
		uris, _, err := conn.StickerFind("", stickerAlbumFavorite)
		if err != nil {
			if isNoStickerError(err) {
				return nil
			}
			return err
		}

		for _, uri := range uris {
			// Get track info to extract album ID
			attrs, err := conn.ListAllInfo(uri)
			if err != nil || len(attrs) == 0 {
				continue
			}
			albumName := attrs[0]["Album"]
			if albumName == "" {
				continue
			}
			artist := attrs[0]["AlbumArtist"]
			if artist == "" {
				artist = attrs[0]["Artist"]
			}
			albumID := encodeAlbumID(albumName, artist)
			albumIDs[albumID] = struct{}{}
		}
		return nil
	})

	return albumIDs, err
}

// getFavoriteAlbums returns albums that have been marked as album favorites.
// This looks for the album_favorite sticker, not the track-level favorite sticker.
func (m *mpdMediaProvider) getFavoriteAlbums() ([]*mediaprovider.Album, error) {
	var albums []*mediaprovider.Album

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Find all files with album_favorite sticker
		uris, _, err := conn.StickerFind("", stickerAlbumFavorite)
		if err != nil {
			if isNoStickerError(err) {
				return nil
			}
			return err
		}

		// Collect unique albums
		albumMap := make(map[string]*mediaprovider.Album)
		for _, uri := range uris {
			attrs, err := conn.ListAllInfo(uri)
			if err != nil || len(attrs) == 0 {
				continue
			}
			albumName := attrs[0]["Album"]
			if albumName == "" {
				continue
			}
			artist := attrs[0]["AlbumArtist"]
			if artist == "" {
				artist = attrs[0]["Artist"]
			}
			albumID := encodeAlbumID(albumName, artist)
			if albumMap[albumID] == nil {
				album := toAlbum(albumName, artist, 0, 0)
				if album != nil {
					album.Favorite = true
					albumMap[albumID] = album
				}
			}
		}

		for _, album := range albumMap {
			albums = append(albums, album)
		}
		return nil
	})

	return albums, err
}

// setAlbumFavorite sets the album_favorite sticker for all tracks in an album.
// This is separate from track-level favorites to allow distinguishing between
// "favorited album" and "favorited individual tracks".
func (m *mpdMediaProvider) setAlbumFavorite(albumID string, favorite bool) error {
	albumName, artistName, ok := decodeAlbumID(albumID)
	if !ok {
		return ErrNotSupported
	}

	return m.server.withConn(func(conn *mpd.Client) error {
		// Find all tracks in this album
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

		// Set album_favorite sticker for each track
		for _, a := range attrs {
			file := a["file"]
			if file == "" {
				continue
			}
			if favorite {
				if err := conn.StickerSet(file, stickerAlbumFavorite, "1"); err != nil {
					log.Printf("Error setting album favorite for %s: %v", file, err)
				}
			} else {
				if err := conn.StickerDelete(file, stickerAlbumFavorite); err != nil && !isNoStickerError(err) {
					log.Printf("Error removing album favorite for %s: %v", file, err)
				}
			}
		}
		return nil
	})
}

// setAlbumRating sets the rating sticker for all tracks in an album.
func (m *mpdMediaProvider) setAlbumRating(albumID string, rating int) error {
	albumName, artistName, ok := decodeAlbumID(albumID)
	if !ok {
		return ErrNotSupported
	}

	return m.server.withConn(func(conn *mpd.Client) error {
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

		for _, a := range attrs {
			file := a["file"]
			if file == "" {
				continue
			}
			if rating > 0 && rating <= 5 {
				if err := conn.StickerSet(file, stickerRating, strconv.Itoa(rating)); err != nil {
					log.Printf("Error setting rating for %s: %v", file, err)
				}
			} else {
				if err := conn.StickerDelete(file, stickerRating); err != nil && !isNoStickerError(err) {
					log.Printf("Error removing rating for %s: %v", file, err)
				}
			}
		}
		return nil
	})
}

// setArtistFavorite sets the artist_favorite sticker for all tracks by an artist.
func (m *mpdMediaProvider) setArtistFavorite(artistID string, favorite bool) error {
	artistName, ok := decodeArtistID(artistID)
	if !ok {
		return ErrNotSupported
	}

	return m.server.withConn(func(conn *mpd.Client) error {
		// Find all tracks by this artist (as album artist or track artist)
		attrs, err := conn.Find("albumartist", artistName)
		if err != nil {
			return err
		}

		// Also find tracks where artist (not album artist) matches
		attrs2, err := conn.Find("artist", artistName)
		if err == nil {
			attrs = append(attrs, attrs2...)
		}

		// Deduplicate by file path
		seen := make(map[string]bool)
		for _, a := range attrs {
			file := a["file"]
			if file == "" || seen[file] {
				continue
			}
			seen[file] = true

			if favorite {
				if err := conn.StickerSet(file, stickerArtistFavorite, artistID); err != nil {
					log.Printf("Error setting artist favorite for %s: %v", file, err)
				}
			} else {
				if err := conn.StickerDelete(file, stickerArtistFavorite); err != nil && !isNoStickerError(err) {
					log.Printf("Error removing artist favorite for %s: %v", file, err)
				}
			}
		}
		return nil
	})
}

// getFavoriteArtistIDs returns a set of artist IDs that have been marked as favorites.
func (m *mpdMediaProvider) getFavoriteArtistIDs() (map[string]struct{}, error) {
	artistIDs := make(map[string]struct{})

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Find all files with artist_favorite sticker
		_, stickers, err := conn.StickerFind("", stickerArtistFavorite)
		if err != nil {
			if isNoStickerError(err) {
				return nil
			}
			return err
		}

		// The sticker value contains the artist ID
		for _, sticker := range stickers {
			if sticker.Value != "" {
				artistIDs[sticker.Value] = struct{}{}
			}
		}
		return nil
	})

	return artistIDs, err
}

// getFavoriteArtists returns artists that have been marked as favorites.
func (m *mpdMediaProvider) getFavoriteArtists() ([]*mediaprovider.Artist, error) {
	artistIDs, err := m.getFavoriteArtistIDs()
	if err != nil {
		return nil, err
	}

	var artists []*mediaprovider.Artist
	err = m.server.withConn(func(conn *mpd.Client) error {
		for artistID := range artistIDs {
			artistName, ok := decodeArtistID(artistID)
			if !ok {
				continue
			}

			// Count albums by this artist
			var albumCount int
			albums, err := conn.Find("albumartist", artistName)
			if err == nil && len(albums) > 0 {
				albumSet := make(map[string]bool)
				for _, a := range albums {
					if albumName := a["Album"]; albumName != "" {
						albumSet[albumName] = true
					}
				}
				albumCount = len(albumSet)
			}

			// Note: CoverArtID is left empty for artists - artist images come from
			// external sources (Deezer) via GetArtistInfo, not from embedded covers
			artist := &mediaprovider.Artist{
				ID:         artistID,
				Name:       artistName,
				AlbumCount: albumCount,
				Favorite:   true,
			}
			artists = append(artists, artist)
		}
		return nil
	})

	return artists, err
}

// isNoStickerError checks if an error is a sticker-related error that can be ignored.
// This includes "no such sticker" errors and "stickers not enabled" errors.
func isNoStickerError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "no sticker found" ||
		errStr == "no such sticker" ||
		errStr == "ACK [50@0] {sticker} no such sticker" ||
		// Handle stickers disabled in MPD config
		strings.Contains(errStr, "sticker") ||
		strings.Contains(errStr, "not enabled") ||
		strings.Contains(errStr, "unknown command")
}
