package jellyfin

import (
	"github.com/dweymouth/go-jellyfin"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/mediaprovider/helpers"
	"github.com/dweymouth/supersonic/sharedutil"
)

const (
	ArtistSortNameAZ string = "Name (A-Z)"
)

func (j *jellyfinMediaProvider) ArtistSortOrders() []string {
	return []string{
		ArtistSortNameAZ,
	}
}

func (j *jellyfinMediaProvider) IterateArtists(sortOrder string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	var jfSort jellyfin.Sort

	if sortOrder == "" {
		sortOrder = ArtistSortNameAZ // default
	}
	switch sortOrder {
	case ArtistSortNameAZ:
		jfSort.Field = jellyfin.SortByName
		jfSort.Mode = jellyfin.SortAsc
	}

	fetcher := func(offs, limit int) ([]*mediaprovider.Artist, error) {
		ar, err := j.client.GetAlbumArtists(jellyfin.QueryOpts{
			Sort:   jfSort,
			Paging: jellyfin.Paging{StartIndex: offs, Limit: limit},
		})
		if err != nil {
			return nil, err
		}
		return sharedutil.MapSlice(ar, toArtist), nil
	}

	return helpers.NewArtistIterator(fetcher, filter, j.prefetchCoverCB)
}

func (j *jellyfinMediaProvider) SearchArtists(searchQuery string, filter mediaprovider.ArtistFilter) mediaprovider.ArtistIterator {
	fetcher := func(offs, limit int) ([]*mediaprovider.Artist, error) {
		sr, err := j.client.Search(searchQuery, jellyfin.TypeArtist, jellyfin.Paging{StartIndex: offs, Limit: limit})
		if err != nil {
			return nil, err
		}
		return sharedutil.MapSlice(sr.Artists, toArtist), nil
	}
	return helpers.NewArtistIterator(fetcher, filter, j.prefetchCoverCB)
}
