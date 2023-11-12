package jellyfin

import "github.com/dweymouth/supersonic/backend/mediaprovider"

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

func (s *jellyfinMediaProvider) IterateAlbums(sortOrder string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return nil
	// TODO: unimplemented
}

func (s *jellyfinMediaProvider) SearchAlbums(searchQuery string, filter mediaprovider.AlbumFilter) mediaprovider.AlbumIterator {
	return nil
	// TODO: unimplemented
}

func (s *jellyfinMediaProvider) IterateTracks(searchQuery string) mediaprovider.TrackIterator {
	return nil
	// TODO: unimplemented
}
