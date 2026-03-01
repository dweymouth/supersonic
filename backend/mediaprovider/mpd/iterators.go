package mpd

import (
	"log"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/deluan/sanitize"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/fhs/gompd/v2/mpd"
)

// albumIterator iterates over albums from MPD.
type albumIterator struct {
	provider        *mpdMediaProvider
	albums          []*mediaprovider.Album
	filter          mediaprovider.AlbumFilter
	pos             int
	loaded          bool
	sortOrder       string
	lastRefreshTime time.Time
	loadingComplete bool
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

	// If loading not complete, refresh every 200ms (or immediately if we have
	// no albums yet, so we pick up the cache without waiting for the timer).
	if !it.loadingComplete {
		if len(it.albums) == 0 || time.Since(it.lastRefreshTime) > 200*time.Millisecond {
			it.refreshAlbums()
		}
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

	// Loading is still in progress but we have no more albums to show right now.
	// Return nil so the UI can render what it has; it will call Next() again
	// after a refresh, picking up newly cached albums.
	return nil
}

func (it *albumIterator) loadAlbums() {
	it.loaded = true

	// Retry up to 3 times if database is being updated or connection errors occur
	var albums []*mediaprovider.Album
	var err error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		albums, err = it.provider.getAllAlbums()
		if err == nil {
			break
		}
		// Check if error is due to database update
		if strings.Contains(err.Error(), "database was updated during query") {
			log.Printf("database updated during album query, retrying (%d/%d)...", i+1, maxRetries)
			continue
		}
		// Check for EOF or connection errors - retry with backoff
		if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "connection") {
			log.Printf("connection error during album query, retrying (%d/%d)...", i+1, maxRetries)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
			continue
		}
		// Other error, don't retry
		break
	}
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
			iTime := albumStats[albums[i].ID].lastPlayed
			jTime := albumStats[albums[j].ID].lastPlayed
			// Primary sort by play time
			if !iTime.Equal(jTime) {
				return iTime.After(jTime)
			}
			// Secondary sort by album name for stable ordering
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
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
			iCount := albumStats[albums[i].ID].playCount
			jCount := albumStats[albums[j].ID].playCount
			// Primary sort by play count
			if iCount != jCount {
				return iCount > jCount
			}
			// Secondary sort by album name for stable ordering
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
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
			// Primary sort by artist name
			if aArtist != bArtist {
				return aArtist < bArtist
			}
			// Secondary sort by album name for stable ordering
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
		})
	case mediaprovider.AlbumSortYearAscending:
		sort.Slice(albums, func(i, j int) bool {
			iYear := albums[i].YearOrZero()
			jYear := albums[j].YearOrZero()
			// Primary sort by year ascending
			if iYear != jYear {
				return iYear < jYear
			}
			// Secondary sort by album name for stable ordering
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
		})
	case mediaprovider.AlbumSortYearDescending:
		sort.Slice(albums, func(i, j int) bool {
			iYear := albums[i].YearOrZero()
			jYear := albums[j].YearOrZero()
			// Primary sort by year descending
			if iYear != jYear {
				return iYear > jYear
			}
			// Secondary sort by album name for stable ordering
			return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name)
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
	it.lastRefreshTime = time.Now()
	it.loadingComplete = it.provider.isAlbumLoadingComplete()
}

// Done returns true only when the background loader has finished AND all
// loaded albums have been yielded. This satisfies the widgets.iteratorDone
// interface so the GridView knows when to stop polling for more items.
func (it *albumIterator) Done() bool {
	return it.loadingComplete && it.pos >= len(it.albums)
}

// refreshAlbums checks if more albums are available and appends new ones.
// IMPORTANT: we never replace it.albums wholesale because it.pos is an index
// into that slice — replacing it would make pos point to wrong items and cause
// visible reordering in the UI. Instead we only append albums not already present.
func (it *albumIterator) refreshAlbums() {
	freshAlbums, err := it.provider.getAllAlbums()
	if err != nil {
		return
	}

	if len(freshAlbums) <= len(it.albums) {
		it.lastRefreshTime = time.Now()
		it.loadingComplete = it.provider.isAlbumLoadingComplete()
		return
	}

	// Build a set of IDs already in our list.
	knownIDs := make(map[string]struct{}, len(it.albums))
	for _, a := range it.albums {
		knownIDs[a.ID] = struct{}{}
	}

	favoriteAlbumIDs, _ := it.provider.getFavoriteAlbumIDs()

	var newAlbums []*mediaprovider.Album
	for _, a := range freshAlbums {
		if _, seen := knownIDs[a.ID]; seen {
			continue
		}
		if favoriteAlbumIDs != nil {
			if _, fav := favoriteAlbumIDs[a.ID]; fav {
				a.Favorite = true
			}
		}
		newAlbums = append(newAlbums, a)
	}

	if len(newAlbums) == 0 {
		it.lastRefreshTime = time.Now()
		it.loadingComplete = it.provider.isAlbumLoadingComplete()
		return
	}

	// Sort only the new arrivals and append them — never reorder existing items.
	switch it.sortOrder {
	case mediaprovider.AlbumSortRecentlyPlayed:
		albumStats := it.provider.getAlbumPlayStats(newAlbums)
		sort.Slice(newAlbums, func(i, j int) bool {
			iTime := albumStats[newAlbums[i].ID].lastPlayed
			jTime := albumStats[newAlbums[j].ID].lastPlayed
			if !iTime.Equal(jTime) {
				return iTime.After(jTime)
			}
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
		// Only append albums that have been played.
		var played []*mediaprovider.Album
		for _, a := range newAlbums {
			if !albumStats[a.ID].lastPlayed.IsZero() {
				played = append(played, a)
			}
		}
		newAlbums = played
	case mediaprovider.AlbumSortFrequentlyPlayed:
		albumStats := it.provider.getAlbumPlayStats(newAlbums)
		sort.Slice(newAlbums, func(i, j int) bool {
			iCount := albumStats[newAlbums[i].ID].playCount
			jCount := albumStats[newAlbums[j].ID].playCount
			if iCount != jCount {
				return iCount > jCount
			}
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
		var played []*mediaprovider.Album
		for _, a := range newAlbums {
			if albumStats[a.ID].playCount > 0 {
				played = append(played, a)
			}
		}
		newAlbums = played
	case mediaprovider.AlbumSortTitleAZ:
		sort.Slice(newAlbums, func(i, j int) bool {
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
	case mediaprovider.AlbumSortArtistAZ:
		sort.Slice(newAlbums, func(i, j int) bool {
			aArtist, bArtist := "", ""
			if len(newAlbums[i].ArtistNames) > 0 {
				aArtist = strings.ToLower(newAlbums[i].ArtistNames[0])
			}
			if len(newAlbums[j].ArtistNames) > 0 {
				bArtist = strings.ToLower(newAlbums[j].ArtistNames[0])
			}
			if aArtist != bArtist {
				return aArtist < bArtist
			}
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
	case mediaprovider.AlbumSortYearAscending:
		sort.Slice(newAlbums, func(i, j int) bool {
			iYear, jYear := newAlbums[i].YearOrZero(), newAlbums[j].YearOrZero()
			if iYear != jYear {
				return iYear < jYear
			}
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
	case mediaprovider.AlbumSortYearDescending:
		sort.Slice(newAlbums, func(i, j int) bool {
			iYear, jYear := newAlbums[i].YearOrZero(), newAlbums[j].YearOrZero()
			if iYear != jYear {
				return iYear > jYear
			}
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
	case mediaprovider.AlbumSortRandom:
		// Append in cache arrival order.
	default:
		sort.Slice(newAlbums, func(i, j int) bool {
			return strings.ToLower(newAlbums[i].Name) < strings.ToLower(newAlbums[j].Name)
		})
	}

	it.albums = append(it.albums, newAlbums...)
	it.lastRefreshTime = time.Now()
	it.loadingComplete = it.provider.isAlbumLoadingComplete()
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
					// If no tracks found with albumartist, try with regular artist tag as fallback
					if len(albumAttrs) == 0 {
						albumAttrs, _ = conn.Find("album", albumName, "artist", artist)
					}
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
	provider        *mpdMediaProvider
	sortOrder       string
	filter          mediaprovider.ArtistFilter
	artists         []*mediaprovider.Artist
	pos             int
	loaded          bool
	lastRefreshTime time.Time
	loadingComplete bool
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

	// Check if we should refresh. If we have no artists yet, bypass the timer
	// so we pick up the cache on the very next call rather than waiting 200ms.
	if !it.loadingComplete {
		if len(it.artists) == 0 || time.Since(it.lastRefreshTime) > 200*time.Millisecond {
			it.refreshArtists()
		}
	}

	for it.pos < len(it.artists) {
		artist := it.artists[it.pos]
		it.pos++
		if it.filter.Matches(artist) {
			return artist
		}
	}

	// Loading is still in progress but we have no more artists to show right now.
	// Return nil so the UI can render what it has; it will call Next() again
	// after a refresh, picking up newly cached artists.
	return nil
}

func (it *artistIterator) loadArtists() {
	it.loaded = true

	// Get current artists from cache
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
			// Primary sort by album count descending
			if artists[i].AlbumCount != artists[j].AlbumCount {
				return artists[i].AlbumCount > artists[j].AlbumCount
			}
			// Secondary sort by name ascending for stable ordering
			return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name)
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
	it.lastRefreshTime = time.Now()
	it.loadingComplete = it.provider.isArtistLoadingComplete()
}

// Done returns true only when the background loader has finished AND all
// loaded artists have been yielded. This satisfies the widgets.iteratorDone
// interface so the GridView knows when to stop polling for more items.
func (it *artistIterator) Done() bool {
	return it.loadingComplete && it.pos >= len(it.artists)
}

// refreshArtists checks if more artists are available and appends new ones.
// IMPORTANT: we never replace it.artists wholesale because it.pos is an index
// into that slice — replacing it would make pos point to wrong items and cause
// visible reordering in the UI. Instead we only append artists that are not
// already present.
func (it *artistIterator) refreshArtists() {
	// Get current artists from cache
	freshArtists, err := it.provider.getAllArtists()
	if err != nil {
		return // Keep existing artists on error
	}

	if len(freshArtists) <= len(it.artists) {
		it.lastRefreshTime = time.Now()
		it.loadingComplete = it.provider.isArtistLoadingComplete()
		return
	}

	// Build a set of IDs already in our list so we can find newcomers.
	knownIDs := make(map[string]struct{}, len(it.artists))
	for _, a := range it.artists {
		knownIDs[a.ID] = struct{}{}
	}

	// Apply favorites to new artists only.
	favoriteArtistIDs, _ := it.provider.getFavoriteArtistIDs()

	var newArtists []*mediaprovider.Artist
	for _, a := range freshArtists {
		if _, seen := knownIDs[a.ID]; seen {
			continue
		}
		if favoriteArtistIDs != nil {
			if _, fav := favoriteArtistIDs[a.ID]; fav {
				a.Favorite = true
			}
		}
		newArtists = append(newArtists, a)
	}

	if len(newArtists) == 0 {
		it.lastRefreshTime = time.Now()
		it.loadingComplete = it.provider.isArtistLoadingComplete()
		return
	}

	// Sort only the new arrivals (random order: append as-is to preserve initial shuffle).
	switch it.sortOrder {
	case mediaprovider.ArtistSortAlbumCount:
		sort.Slice(newArtists, func(i, j int) bool {
			if newArtists[i].AlbumCount != newArtists[j].AlbumCount {
				return newArtists[i].AlbumCount > newArtists[j].AlbumCount
			}
			return strings.ToLower(newArtists[i].Name) < strings.ToLower(newArtists[j].Name)
		})
	case mediaprovider.ArtistSortRandom:
		// Append in cache order (random-ish arrival order is fine).
	default: // ArtistSortNameAZ
		sort.Slice(newArtists, func(i, j int) bool {
			return strings.ToLower(newArtists[i].Name) < strings.ToLower(newArtists[j].Name)
		})
	}

	// Append to the end — never touch items before it.pos.
	it.artists = append(it.artists, newArtists...)
	it.lastRefreshTime = time.Now()
	it.loadingComplete = it.provider.isArtistLoadingComplete()
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

	// Get all artists from cache (background loading)
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
	it.artists = artists
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
