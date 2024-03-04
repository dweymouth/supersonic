package subsonic

import (
	"cmp"
	"log"
	"slices"
	"strings"

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

func (s *subsonicMediaProvider) IterateArtists(sortOrder string) mediaprovider.ArtistIterator {
	if sortOrder == "" {
		sortOrder = ArtistSortNameAZ // default
	}
	switch sortOrder {
	case ArtistSortNameAZ:
		return s.baseArtistIterFromSimpleSortOrder(
			func(artists []*subsonic.ArtistID3) []*subsonic.ArtistID3 {
				slices.SortFunc(artists, func(a, b *subsonic.ArtistID3) int {
					return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
				})
				return artists
			},
		)
	default:
		log.Printf("Undefined artist sort order: %s", sortOrder)
		return nil
	}
}

func (s *subsonicMediaProvider) baseArtistIterFromSimpleSortOrder(sortFn func([]*subsonic.ArtistID3) []*subsonic.ArtistID3) mediaprovider.ArtistIterator {
	return helpers.NewArtistIterator(s.artistFetchFnFromStandardSort(sortFn))
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
