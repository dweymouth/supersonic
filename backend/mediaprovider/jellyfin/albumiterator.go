package jellyfin

import (
	"github.com/dweymouth/go-jellyfin"
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

func (j *jellyfinMediaProvider) AlbumSortOrders() []string {
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

func (j *jellyfinMediaProvider) IterateAlbums(sortOrder string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	var jfFilt jellyfin.Filter
	if filter.ExcludeUnfavorited {
		jfFilt.Favorite = true
		filter.ExcludeUnfavorited = false
	}
	var jfSort jellyfin.Sort
	switch sortOrder {
	case AlbumSortRecentlyAdded:
		jfSort.Field = "DateCreated"
		jfSort.Mode = jellyfin.SortDesc
	case AlbumSortFrequentlyPlayed:
		jfSort.Field = "PlayCount"
		jfSort.Mode = jellyfin.SortDesc
	case AlbumSortRandom:
		jfSort.Field = "Random"
	case AlbumSortArtistAZ:
		jfSort.Field = "AlbumArtist"
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortTitleAZ:
		jfSort.Field = "SortName"
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortRecentlyPlayed:
		jfSort.Field = "DatePlayed"
		jfSort.Mode = jellyfin.SortDesc
	case AlbumSortYearAscending:
		jfSort.Field = "ProductionYear"
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortYearDescending:
		jfSort.Field = "ProductionYear"
		jfSort.Mode = jellyfin.SortDesc
	}
	fetcher := func(offs, limit int) ([]*mediaprovider.Album, error) {
		al, err := j.client.GetAlbums(jellyfin.QueryOpts{
			Sort:   jfSort,
			Filter: jfFilt,
			Paging: jellyfin.Paging{StartIndex: offs, Limit: limit},
		})
		if err != nil {
			return nil, err
		}
		return sharedutil.MapSlice(al, toAlbum), nil
	}
	if sortOrder == AlbumSortRandom {
		determFetcher := func(offs, limit int) ([]*mediaprovider.Album, error) {
			al, err := j.client.GetAlbums(jellyfin.QueryOpts{
				Sort:   jellyfin.Sort{Field: "SortName", Mode: jellyfin.SortAsc},
				Filter: jfFilt,
				Paging: jellyfin.Paging{StartIndex: offs, Limit: limit},
			})
			if err != nil {
				return nil, err
			}
			return sharedutil.MapSlice(al, toAlbum), nil
		}
		return helpers.NewRandomIter(determFetcher, fetcher, filter, j.prefetchCoverCB)
	}
	return helpers.NewBaseIter(fetcher, filter, j.prefetchCoverCB)
}

func (s *jellyfinMediaProvider) SearchAlbums(searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return nil
	// TODO: unimplemented
}

func (s *jellyfinMediaProvider) IterateTracks(searchQuery string) mediaprovider.TrackIterator {
	return nil
	// TODO: unimplemented
}
