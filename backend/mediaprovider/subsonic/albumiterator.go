package subsonic

import (
	"log"
	"strconv"
	"strings"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/mediaprovider/helpers"
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

func filterMatches(f mediaprovider.AlbumFilter, album *subsonic.AlbumID3, ignoreGenre bool) bool {
	if album == nil {
		return false
	}
	if f.ExcludeFavorited && !album.Starred.IsZero() {
		return false
	}
	if f.ExcludeUnfavorited && album.Starred.IsZero() {
		return false
	}
	if y := album.Year; y < f.MinYear || (f.MaxYear > 0 && y > f.MaxYear) {
		return false
	}
	if ignoreGenre || len(f.Genres) == 0 {
		return true
	}
	for _, g := range f.Genres {
		if strings.EqualFold(g, album.Genre) {
			return true
		}
	}
	return false
}

func (s *subsonicMediaProvider) IterateAlbums(sortOrder string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	if sortOrder == "" && len(filter.Genres) == 1 {
		return s.newBaseIter("byGenre", filter, s.prefetchCoverCB, map[string]string{"genre": filter.Genres[0]})
	}
	if sortOrder == "" && filter.ExcludeUnfavorited {
		return s.newBaseIter("starred", filter, s.prefetchCoverCB, make(map[string]string))
	}
	if sortOrder == "" {
		sortOrder = AlbumSortRecentlyAdded // default
	}
	switch sortOrder {
	case AlbumSortRecentlyAdded:
		return s.newBaseIter("newest", filter, s.prefetchCoverCB, make(map[string]string))
	case AlbumSortRecentlyPlayed:
		return s.newBaseIter("recent", filter, s.prefetchCoverCB, make(map[string]string))
	case AlbumSortFrequentlyPlayed:
		return s.newBaseIter("frequent", filter, s.prefetchCoverCB, make(map[string]string))
	case AlbumSortRandom:
		return s.newRandomIter(filter, s.prefetchCoverCB)
	case AlbumSortTitleAZ:
		return s.newBaseIter("alphabeticalByName", filter, s.prefetchCoverCB, make(map[string]string))
	case AlbumSortArtistAZ:
		return s.newBaseIter("alphabeticalByArtist", filter, s.prefetchCoverCB, make(map[string]string))
	case AlbumSortYearAscending:
		return s.newBaseIter("byYear", filter, s.prefetchCoverCB, map[string]string{"fromYear": "0", "toYear": "3000"})
	case AlbumSortYearDescending:
		return s.newBaseIter("byYear", filter, s.prefetchCoverCB, map[string]string{"fromYear": "3000", "toYear": "0"})
	default:
		log.Printf("Undefined album sort order: %s", sortOrder)
		return nil
	}
}

func (s *subsonicMediaProvider) SearchAlbums(searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return s.newSearchIter(searchQuery, filter, s.prefetchCoverCB)
}

type baseIter struct {
	listType      string
	filter        mediaprovider.AlbumFilter
	prefetchCB    func(string)
	serverPos     int
	s             *subsonic.Client
	opts          map[string]string
	prefetched    []*mediaprovider.Album
	prefetchedPos int
	done          bool
}

func (s *subsonicMediaProvider) newBaseIter(listType string, filter mediaprovider.AlbumFilter, cb func(string), opts map[string]string) *baseIter {
	return &baseIter{
		prefetchCB: cb,
		listType:   listType,
		filter:     filter,
		s:          s.client,
		opts:       opts,
	}
}

func (r *baseIter) Next() *mediaprovider.Album {
	if r.done {
		return nil
	}
	if r.prefetched != nil && r.prefetchedPos < len(r.prefetched) {
		a := r.prefetched[r.prefetchedPos]
		r.prefetchedPos++
		return a
	}
	r.prefetched = nil
	for { // keep fetching until we are done or have mathcing results
		r.opts["offset"] = strconv.Itoa(r.serverPos)
		albums, err := r.s.GetAlbumList2(r.listType, r.opts)
		if err != nil {
			log.Printf("error fetching albums: %s", err.Error())
			albums = nil
		}
		if len(albums) == 0 {
			r.done = true
			return nil
		}
		r.serverPos += len(albums)
		albums = sharedutil.FilterSlice(albums, func(al *subsonic.AlbumID3) bool {
			// The Subsonic API returns only the first genre for multi-genre albums,
			// but servers do internally match against all the genres the album is categorized with.
			// So we must not additionally filter by genre to avoid excluding results where
			// the single genre returned by Subsonic isn't the one we're iterating on.
			return filterMatches(r.filter, al, r.listType == "byGenre" /*ignoreGenre*/)
		})
		r.prefetched = sharedutil.MapSlice(albums, toAlbum)
		if len(albums) > 0 {
			break
		}
	}
	r.prefetchedPos = 1
	if r.prefetchCB != nil {
		for _, album := range r.prefetched {
			go r.prefetchCB(album.CoverArtID)
		}
	}
	return r.prefetched[0]
}

type searchIter struct {
	searchIterBase

	prefetchCB    func(string)
	filter        mediaprovider.AlbumFilter
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	albumIDset    map[string]bool
	done          bool
}

func (s *subsonicMediaProvider) newSearchIter(query string, filter mediaprovider.AlbumFilter, cb func(string)) *searchIter {
	return &searchIter{
		searchIterBase: searchIterBase{
			query: query,
			s:     s.client,
		},
		prefetchCB: cb,
		filter:     filter,
		albumIDset: make(map[string]bool),
	}
}

func (s *searchIter) Next() *mediaprovider.Album {
	if s.done {
		return nil
	}

	// prefetch more search results from server
	if s.prefetched == nil {
		results := s.searchIterBase.fetchResults()
		if results == nil {
			s.done = true
			s.albumIDset = nil
			return nil
		}

		// add results from albums search
		s.addNewAlbums(results.Album)
		s.albumOffset += len(results.Album)

		// add results from artists search
		for _, artist := range results.Artist {
			artist, err := s.s.GetArtist(artist.ID)
			if err != nil || artist == nil {
				log.Printf("error fetching artist: %s", err.Error())
			} else {
				s.addNewAlbums(artist.Album)
			}
		}
		s.artistOffset += len(results.Artist)

		// add results from songs search
		for _, song := range results.Song {
			if song.AlbumID == "" {
				continue
			}
			album, err := s.s.GetAlbum(song.AlbumID)
			if err != nil || album == nil {
				log.Printf("error fetching album: %s", err.Error())
			} else {
				s.addNewAlbums([]*subsonic.AlbumID3{album})
			}
		}
		s.songOffset += len(results.Song)
	}

	// return from prefetched results
	if len(s.prefetched) > 0 {
		a := s.prefetched[s.prefetchedPos]
		s.prefetchedPos++
		if s.prefetchedPos == len(s.prefetched) {
			s.prefetched = nil
			s.prefetchedPos = 0
		}

		return toAlbum(a)
	}

	return nil
}

func (s *searchIter) addNewAlbums(al []*subsonic.AlbumID3) {
	for _, album := range al {
		if _, have := s.albumIDset[album.ID]; have {
			continue
		}
		if !filterMatches(s.filter, album, false) {
			continue
		}
		s.prefetched = append(s.prefetched, album)
		if s.prefetchCB != nil {
			go s.prefetchCB(album.CoverArt)
		}
		s.albumIDset[album.ID] = true
	}
}

type randomIter struct {
	filter        mediaprovider.AlbumFilter
	prefetchCB    func(coverArtID string)
	albumIDSet    map[string]bool
	s             *subsonic.Client
	prefetched    []*subsonic.AlbumID3
	prefetchedPos int
	// Random iter works in two phases - phase 1 by requesting random
	// albums from the server. Since the Subsonic API provides no way
	// of paginating a single random sort, we may get albums back twice.
	// We use albumIDSet to keep track of which albums have already been returned.
	// Once we start getting back too many already-returned albums,
	// switch to requesting more albums from a deterministic sort order.
	phaseTwo bool
	offset   int
	done     bool
}

func (s *subsonicMediaProvider) newRandomIter(filter mediaprovider.AlbumFilter, cb func(string)) mediaprovider.AlbumIterator {
	return helpers.NewRandomIter(
		func(offset, limit int) ([]*mediaprovider.Album, error) {
			al, err := s.client.GetAlbumList2("newest", map[string]string{"size": strconv.Itoa(limit), "offset": strconv.Itoa(offset)})
			if err != nil {
				return nil, err
			}
			return sharedutil.MapSlice(al, toAlbum), nil
		},
		func(_, limit int) ([]*mediaprovider.Album, error) {
			al, err := s.client.GetAlbumList2("random", map[string]string{"size": strconv.Itoa(limit)})
			if err != nil {
				return nil, err
			}
			return sharedutil.MapSlice(al, toAlbum), nil
		},
		filter, s.prefetchCoverCB)
}
