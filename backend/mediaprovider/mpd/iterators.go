package mpd

import (
	"log"
	"math/rand"
	"sort"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
	"github.com/fhs/gompd/v2/mpd"
)

// albumIterator iterates over albums from MPD.
type albumIterator struct {
	provider  *mpdMediaProvider
	albums    []*mediaprovider.Album
	filter    mediaprovider.AlbumFilter
	pos       int
	loaded    bool
	sortOrder string
}

func newAlbumIterator(provider *mpdMediaProvider, sortOrder string, filter mediaprovider.AlbumFilter) *albumIterator {
	return &albumIterator{
		provider:  provider,
		sortOrder: sortOrder,
		filter:    filter,
	}
}

func (it *albumIterator) Next() *mediaprovider.Album {
	if !it.loaded {
		it.loadAlbums()
	}

	for it.pos < len(it.albums) {
		album := it.albums[it.pos]
		it.pos++
		if it.filter.Matches(album) {
			if it.provider.prefetchCoverCB != nil {
				go it.provider.prefetchCoverCB(album.CoverArtID)
			}
			return album
		}
	}
	return nil
}

func (it *albumIterator) loadAlbums() {
	it.loaded = true

	albums, err := it.provider.getAllAlbums()
	if err != nil {
		log.Printf("error loading albums: %v", err)
		return
	}

	// Mark favorite albums based on stickers
	favoriteAlbumIDs, err := it.provider.getFavoriteAlbumIDs()
	if err != nil {
		log.Printf("error getting favorite album IDs: %v", err)
		// Continue without favorites - not a fatal error
	} else {
		for _, album := range albums {
			if _, ok := favoriteAlbumIDs[album.ID]; ok {
				album.Favorite = true
			}
		}
	}

	// Sort albums based on sort order
	switch it.sortOrder {
	case mediaprovider.AlbumSortRecentlyPlayed:
		// Sort by most recent play time (descending)
		albumStats := it.provider.getAlbumPlayStats(albums)
		sort.Slice(albums, func(i, j int) bool {
			return albumStats[albums[i].ID].lastPlayed.After(albumStats[albums[j].ID].lastPlayed)
		})
		// Filter to only albums that have been played
		var playedAlbums []*mediaprovider.Album
		for _, album := range albums {
			if !albumStats[album.ID].lastPlayed.IsZero() {
				playedAlbums = append(playedAlbums, album)
			}
		}
		albums = playedAlbums

	case mediaprovider.AlbumSortFrequentlyPlayed:
		// Sort by total play count (descending)
		albumStats := it.provider.getAlbumPlayStats(albums)
		sort.Slice(albums, func(i, j int) bool {
			return albumStats[albums[i].ID].playCount > albumStats[albums[j].ID].playCount
		})
		// Filter to only albums that have been played
		var playedAlbums []*mediaprovider.Album
		for _, album := range albums {
			if albumStats[album.ID].playCount > 0 {
				playedAlbums = append(playedAlbums, album)
			}
		}
		albums = playedAlbums

	case mediaprovider.AlbumSortTitleAZ:
		sort.Slice(albums, func(i, j int) bool {
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
		})
	case mediaprovider.AlbumSortArtistAZ:
		sort.Slice(albums, func(i, j int) bool {
			aArtist := ""
			if len(albums[i].ArtistNames) > 0 {
				aArtist = strings.ToLower(albums[i].ArtistNames[0])
			}
			bArtist := ""
			if len(albums[j].ArtistNames) > 0 {
				bArtist = strings.ToLower(albums[j].ArtistNames[0])
			}
			return aArtist < bArtist
		})
	case mediaprovider.AlbumSortYearAscending:
		sort.Slice(albums, func(i, j int) bool {
			return albums[i].YearOrZero() < albums[j].YearOrZero()
		})
	case mediaprovider.AlbumSortYearDescending:
		sort.Slice(albums, func(i, j int) bool {
			return albums[i].YearOrZero() > albums[j].YearOrZero()
		})
	case mediaprovider.AlbumSortRandom:
		// Shuffle albums using Fisher-Yates
		rand.Shuffle(len(albums), func(i, j int) {
			albums[i], albums[j] = albums[j], albums[i]
		})
	default:
		// Default to title A-Z
		sort.Slice(albums, func(i, j int) bool {
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
		})
	}

	it.albums = albums
}

// searchAlbumIterator searches for albums matching a query.
type searchAlbumIterator struct {
	provider   *mpdMediaProvider
	query      string
	filter     mediaprovider.AlbumFilter
	albums     []*mediaprovider.Album
	pos        int
	loaded     bool
	queryTerms []string
}

func newSearchAlbumIterator(provider *mpdMediaProvider, query string, filter mediaprovider.AlbumFilter) *searchAlbumIterator {
	return &searchAlbumIterator{
		provider:   provider,
		query:      query,
		filter:     filter,
		queryTerms: strings.Fields(strings.ToLower(sanitize.Accents(query))),
	}
}

func (it *searchAlbumIterator) Next() *mediaprovider.Album {
	if !it.loaded {
		it.loadAlbums()
	}

	for it.pos < len(it.albums) {
		album := it.albums[it.pos]
		it.pos++
		if it.matchesQuery(album) && it.filter.Matches(album) {
			if it.provider.prefetchCoverCB != nil {
				go it.provider.prefetchCoverCB(album.CoverArtID)
			}
			return album
		}
	}
	return nil
}

func (it *searchAlbumIterator) loadAlbums() {
	it.loaded = true

	// Search for tracks matching the query and collect their albums
	err := it.provider.server.withConn(func(conn *mpd.Client) error {
		attrs, err := conn.Search("any", it.query)
		if err != nil {
			return err
		}

		albumMap := make(map[string]*mediaprovider.Album)
		for _, a := range attrs {
			albumName := a["Album"]
			if albumName == "" {
				continue
			}
			artist := a["AlbumArtist"]
			if artist == "" {
				artist = a["Artist"]
			}
			albumID := encodeAlbumID(albumName, artist)
			if _, exists := albumMap[albumID]; !exists {
				// Get full album info
				var albumAttrs []mpd.Attrs
				if artist != "" {
					albumAttrs, _ = conn.Find("album", albumName, "albumartist", artist)
				} else {
					albumAttrs, _ = conn.Find("album", albumName)
				}
				album := albumFromTracks(albumName, artist, albumAttrs)
				if album != nil {
					albumMap[albumID] = album
				}
			}
		}

		for _, album := range albumMap {
			it.albums = append(it.albums, album)
		}
		return nil
	})

	if err != nil {
		log.Printf("error searching albums: %v", err)
	}

	// Mark favorite albums based on stickers
	favoriteAlbumIDs, err := it.provider.getFavoriteAlbumIDs()
	if err != nil {
		log.Printf("error getting favorite album IDs: %v", err)
	} else {
		for _, album := range it.albums {
			if _, ok := favoriteAlbumIDs[album.ID]; ok {
				album.Favorite = true
			}
		}
	}

	// Sort by relevance (simple name matching)
	sort.Slice(it.albums, func(i, j int) bool {
		aName := strings.ToLower(sanitize.Accents(it.albums[i].Name))
		bName := strings.ToLower(sanitize.Accents(it.albums[j].Name))
		aMatch := strings.Contains(aName, it.query)
		bMatch := strings.Contains(bName, it.query)
		if aMatch != bMatch {
			return aMatch
		}
		return aName < bName
	})
}

func (it *searchAlbumIterator) matchesQuery(album *mediaprovider.Album) bool {
	name := strings.ToLower(sanitize.Accents(album.Name))
	artistStr := strings.ToLower(sanitize.Accents(strings.Join(album.ArtistNames, " ")))
	combined := name + " " + artistStr

	for _, term := range it.queryTerms {
		if !strings.Contains(combined, term) {
			return false
		}
	}
	return true
}

// trackIterator iterates over tracks from MPD.
type trackIterator struct {
	provider *mpdMediaProvider
	query    string
	tracks   []*mediaprovider.Track
	pos      int
	loaded   bool
}

func newTrackIterator(provider *mpdMediaProvider, query string) *trackIterator {
	return &trackIterator{
		provider: provider,
		query:    query,
	}
}

func (it *trackIterator) Next() *mediaprovider.Track {
	if !it.loaded {
		it.loadTracks()
	}

	if it.pos < len(it.tracks) {
		track := it.tracks[it.pos]
		it.pos++
		if it.provider.prefetchCoverCB != nil {
			go it.provider.prefetchCoverCB(track.CoverArtID)
		}
		return track
	}
	return nil
}

func (it *trackIterator) loadTracks() {
	it.loaded = true

	err := it.provider.server.withConn(func(conn *mpd.Client) error {
		var attrs []mpd.Attrs
		var err error
		if it.query != "" {
			attrs, err = conn.Search("any", it.query)
		} else {
			attrs, err = conn.ListAllInfo("/")
		}
		if err != nil {
			return err
		}

		// Filter to actual files
		for _, a := range attrs {
			if a["file"] != "" {
				if track := toTrack(a); track != nil {
					it.tracks = append(it.tracks, track)
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("error loading tracks: %v", err)
	}
}

// artistIterator iterates over artists from MPD.
type artistIterator struct {
	provider  *mpdMediaProvider
	sortOrder string
	filter    mediaprovider.ArtistFilter
	artists   []*mediaprovider.Artist
	pos       int
	loaded    bool
}

func newArtistIterator(provider *mpdMediaProvider, sortOrder string, filter mediaprovider.ArtistFilter) *artistIterator {
	return &artistIterator{
		provider:  provider,
		sortOrder: sortOrder,
		filter:    filter,
	}
}

func (it *artistIterator) Next() *mediaprovider.Artist {
	if !it.loaded {
		it.loadArtists()
	}

	for it.pos < len(it.artists) {
		artist := it.artists[it.pos]
		it.pos++
		if it.filter.Matches(artist) {
			return artist
		}
	}
	return nil
}

func (it *artistIterator) loadArtists() {
	it.loaded = true

	artists, err := it.provider.getAllArtists()
	if err != nil {
		log.Printf("error loading artists: %v", err)
		return
	}

	// Mark favorite artists based on stickers
	favoriteArtistIDs, err := it.provider.getFavoriteArtistIDs()
	if err != nil {
		log.Printf("error getting favorite artist IDs: %v", err)
	} else {
		for _, artist := range artists {
			if _, ok := favoriteArtistIDs[artist.ID]; ok {
				artist.Favorite = true
			}
		}
	}

	// Sort artists
	switch it.sortOrder {
	case mediaprovider.ArtistSortAlbumCount:
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].AlbumCount > artists[j].AlbumCount
		})
	case mediaprovider.ArtistSortRandom:
		rand.Shuffle(len(artists), func(i, j int) {
			artists[i], artists[j] = artists[j], artists[i]
		})
	default: // ArtistSortNameAZ
		sort.Slice(artists, func(i, j int) bool {
			return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name)
		})
	}

	it.artists = artists
}

// searchArtistIterator searches for artists matching a query.
type searchArtistIterator struct {
	provider   *mpdMediaProvider
	query      string
	filter     mediaprovider.ArtistFilter
	artists    []*mediaprovider.Artist
	pos        int
	loaded     bool
	queryTerms []string
}

func newSearchArtistIterator(provider *mpdMediaProvider, query string, filter mediaprovider.ArtistFilter) *searchArtistIterator {
	return &searchArtistIterator{
		provider:   provider,
		query:      query,
		filter:     filter,
		queryTerms: strings.Fields(strings.ToLower(sanitize.Accents(query))),
	}
}

func (it *searchArtistIterator) Next() *mediaprovider.Artist {
	if !it.loaded {
		it.loadArtists()
	}

	for it.pos < len(it.artists) {
		artist := it.artists[it.pos]
		it.pos++
		if it.matchesQuery(artist) && it.filter.Matches(artist) {
			return artist
		}
	}
	return nil
}

func (it *searchArtistIterator) loadArtists() {
	it.loaded = true

	// Get all artists and filter by query
	artists, err := it.provider.getAllArtists()
	if err != nil {
		log.Printf("error loading artists: %v", err)
		return
	}

	// Mark favorite artists based on stickers
	favoriteArtistIDs, err := it.provider.getFavoriteArtistIDs()
	if err != nil {
		log.Printf("error getting favorite artist IDs: %v", err)
	} else {
		for _, artist := range artists {
			if _, ok := favoriteArtistIDs[artist.ID]; ok {
				artist.Favorite = true
			}
		}
	}

	// Filter and sort by relevance
	queryLower := strings.ToLower(sanitize.Accents(it.query))
	it.artists = sharedutil.FilterSlice(artists, func(a *mediaprovider.Artist) bool {
		name := strings.ToLower(sanitize.Accents(a.Name))
		return strings.Contains(name, queryLower)
	})

	// Sort by relevance
	sort.Slice(it.artists, func(i, j int) bool {
		aName := strings.ToLower(sanitize.Accents(it.artists[i].Name))
		bName := strings.ToLower(sanitize.Accents(it.artists[j].Name))
		// Prefer exact matches
		aExact := strings.HasPrefix(aName, queryLower)
		bExact := strings.HasPrefix(bName, queryLower)
		if aExact != bExact {
			return aExact
		}
		return aName < bName
	})
}

func (it *searchArtistIterator) matchesQuery(artist *mediaprovider.Artist) bool {
	name := strings.ToLower(sanitize.Accents(artist.Name))
	for _, term := range it.queryTerms {
		if !strings.Contains(name, term) {
			return false
		}
	}
	return true
}
