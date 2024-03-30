package subsonic

import (
	"log"
	"slices"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/dweymouth/go-subsonic/subsonic"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/mediaprovider/helpers"
	"github.com/dweymouth/supersonic/sharedutil"
)

const (
	ArtistSortNameAZ string = "Name (A-Z)"
)

func (s *subsonicMediaProvider) ArtistSortOrders() []string {
	return []string{
		ArtistSortNameAZ,
	}
}

func filterArtistMatches(f mediaprovider.ArtistFilter, artist *subsonic.ArtistID3) bool {
	if artist == nil {
		return false
	}
	return true
}

func (s *subsonicMediaProvider) IterateArtists(sortOrder string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	if sortOrder == "" {
		sortOrder = ArtistSortNameAZ // default
	}
	switch sortOrder {
	case ArtistSortNameAZ:
		return s.baseArtistIterFromSimpleSortOrder(
			func(artists []*subsonic.ArtistID3) []*subsonic.ArtistID3 {
				c := collate.New(language.English, collate.Loose)
				slices.SortFunc(artists, func(a, b *subsonic.ArtistID3) int {
					return c.CompareString(a.Name, b.Name)
				})
				return artists
			},
			filter,
		)
	default:
		log.Printf("Undefined artist sort order: %s", sortOrder)
		return nil
	}
}

func (s *subsonicMediaProvider) SearchArtists(searchQuery string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	return s.newSearchArtistIter(searchQuery, filter, s.prefetchCoverCB)
}

type searchArtistIter struct {
	searchIterBase

	prefetchCB    func(string)
	filter        mediaprovider.ArtistFilter
	prefetched    []*subsonic.ArtistID3
	prefetchedPos int
	artistIDset   map[string]bool
	done          bool
}

func (s *subsonicMediaProvider) newSearchArtistIter(query string, filter mediaprovider.ArtistFilter, cb func(string)) *searchArtistIter {
	return &searchArtistIter{
		searchIterBase: searchIterBase{
			query: query,
			s:     s.client,
		},
		prefetchCB:  cb,
		filter:      filter,
		artistIDset: make(map[string]bool),
	}
}

func (s *searchArtistIter) Next() *mediaprovider.Artist {
	if s.done {
		return nil
	}

	// prefetch more search results from server
	if s.prefetched == nil {
		results := s.searchIterBase.fetchResults()
		if results == nil {
			s.done = true
			s.artistIDset = nil
			return nil
		}

		// add results from artists search
		s.addNewArtists(results.Artist)
		s.artistOffset += len(results.Artist)
	}

	// return from prefetched results
	if len(s.prefetched) > 0 {
		a := s.prefetched[s.prefetchedPos]
		s.prefetchedPos++
		if s.prefetchedPos == len(s.prefetched) {
			s.prefetched = nil
			s.prefetchedPos = 0
		}

		return toArtistFromID3(a)
	}

	return nil
}

func (s *searchArtistIter) addNewArtists(artists []*subsonic.ArtistID3) {
	for _, artist := range artists {
		if _, have := s.artistIDset[artist.ID]; have {
			continue
		}
		if !filterArtistMatches(s.filter, artist) {
			continue
		}
		s.prefetched = append(s.prefetched, artist)
		if s.prefetchCB != nil {
			go s.prefetchCB(artist.CoverArt)
		}
		s.artistIDset[artist.ID] = true
	}
}

func (s *subsonicMediaProvider) baseArtistIterFromSimpleSortOrder(sortFn func([]*subsonic.ArtistID3) []*subsonic.ArtistID3, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	return helpers.NewArtistIterator(s.artistFetchFnFromStandardSort(sortFn), filter, s.prefetchCoverCB)
}

func (s *subsonicMediaProvider) artistFetchFnFromStandardSort(sortFn func([]*subsonic.ArtistID3) []*subsonic.ArtistID3) helpers.ArtistFetchFn {
	return makeArtistFetchFn(func(offset, limit int) ([]*subsonic.ArtistID3, error) {
		// When the iterator asks for a second page of results, return nil, as Subsonic does not support pagination for artists.
		if offset > 0 {
			return nil, nil
		}

		idxs, err := s.client.GetArtists(map[string]string{})
		if err != nil {
			return nil, err
		}
		var artists []*subsonic.ArtistID3
		for _, idx := range idxs.Index {
			for _, ar := range idx.Artist {
				artists = append(artists, ar)
			}
		}
		artists = sortFn(artists)
		return artists, nil
	})
}

func makeArtistFetchFn(subsonicFetchFn func(offset, limit int) ([]*subsonic.ArtistID3, error)) helpers.ArtistFetchFn {
	return func(offset, limit int) ([]*mediaprovider.Artist, error) {
		ar, err := subsonicFetchFn(offset, limit)
		if err != nil {
			return nil, err
		}
		return sharedutil.MapSlice(ar, toArtistFromID3), nil
	}
}
