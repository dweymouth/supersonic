package jellyfin

import (
	"time"

	"github.com/dweymouth/go-jellyfin"
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/backend/mediaprovider/helpers"
	"github.com/dweymouth/supersonic/sharedutil"
)

const (
	AlbumSortRecentlyAdded  string = "Recently Added"
	AlbumSortRandom         string = "Random"
	AlbumSortTitleAZ        string = "Title (A-Z)"
	AlbumSortArtistAZ       string = "Artist (A-Z)"
	AlbumSortYearAscending  string = "Year (ascending)"
	AlbumSortYearDescending string = "Year (descending)"
)

func (j *jellyfinMediaProvider) AlbumSortOrders() []string {
	return []string{
		AlbumSortRecentlyAdded,
		AlbumSortRandom,
		AlbumSortTitleAZ,
		AlbumSortArtistAZ,
		AlbumSortYearAscending,
		AlbumSortYearDescending,
	}
}

func (j *jellyfinMediaProvider) IterateAlbums(sortOrder string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	var jfSort jellyfin.Sort
	switch sortOrder {
	case AlbumSortRecentlyAdded:
		jfSort.Field = jellyfin.SortByDateCreated
		jfSort.Mode = jellyfin.SortDesc
	case AlbumSortRandom:
		jfSort.Field = jellyfin.SortByRandom
	case AlbumSortArtistAZ:
		jfSort.Field = jellyfin.SortByArtist
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortTitleAZ:
		jfSort.Field = jellyfin.SortByName
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortYearAscending:
		jfSort.Field = jellyfin.SortByYear
		jfSort.Mode = jellyfin.SortAsc
	case AlbumSortYearDescending:
		jfSort.Field = jellyfin.SortByYear
		jfSort.Mode = jellyfin.SortDesc
	}
	jfFilt := jfFilterFromFilter(&filter)

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
		return helpers.NewRandomAlbumIter(determFetcher, fetcher, filter, j.prefetchCoverCB)
	}
	return helpers.NewAlbumIterator(fetcher, filter, j.prefetchCoverCB)
}

func (j *jellyfinMediaProvider) SearchAlbums(searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	fetcher := func(offs, limit int) ([]*mediaprovider.Album, error) {
		sr, err := j.client.Search(searchQuery, jellyfin.TypeAlbum, jellyfin.Paging{StartIndex: offs, Limit: limit})
		if err != nil {
			return nil, err
		}
		return sharedutil.MapSlice(sr.Albums, toAlbum), nil
	}
	return helpers.NewAlbumIterator(fetcher, filter, j.prefetchCoverCB)
}

func (j *jellyfinMediaProvider) IterateTracks(searchQuery string) mediaprovider.TrackIterator {
	var fetcher helpers.TrackFetchFn
	if searchQuery == "" {
		fetcher = func(offs, limit int) ([]*mediaprovider.Track, error) {
			var opts jellyfin.QueryOpts
			opts.Paging = jellyfin.Paging{StartIndex: offs, Limit: limit}
			s, err := j.client.GetSongs(opts)
			if err != nil {
				return nil, err
			}
			return sharedutil.MapSlice(s, toTrack), nil
		}
	} else {
		fetcher = func(offs, limit int) ([]*mediaprovider.Track, error) {
			sr, err := j.client.Search(searchQuery, jellyfin.TypeSong, jellyfin.Paging{StartIndex: offs, Limit: limit})
			if err != nil {
				return nil, err
			}
			return sharedutil.MapSlice(sr.Songs, toTrack), nil
		}
	}
	return helpers.NewTrackIterator(fetcher, j.prefetchCoverCB)
}

// Creates the Jellyfin filter to implement the given mediaprovider filter,
// and zeros out the now-unneeded fields in the mediaprovider filter.
func jfFilterFromFilter(filter *mediaprovider.AlbumFilter) jellyfin.Filter {
	var jfFilt jellyfin.Filter
	if filter.ExcludeUnfavorited {
		jfFilt.Favorite = true
		filter.ExcludeUnfavorited = false // Jellyfin will handle this filter
	}
	if filter.MinYear > 0 && filter.MaxYear > 0 {
		jfFilt.YearRange = [2]int{filter.MinYear, filter.MaxYear}
		filter.MinYear, filter.MaxYear = 0, 0
	} else if filter.MinYear > 0 {
		jfFilt.YearRange = [2]int{filter.MinYear, time.Now().Year()}
		filter.MinYear, filter.MaxYear = 0, 0
	} else if filter.MaxYear > 0 {
		jfFilt.YearRange = [2]int{1900, filter.MaxYear}
		filter.MinYear, filter.MaxYear = 0, 0
	}
	jfFilt.Genres = filter.Genres
	filter.Genres = nil
	return jfFilt
}
