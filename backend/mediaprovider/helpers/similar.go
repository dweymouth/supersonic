package helpers

import (
	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

func GetSimilarSongsFallback(mp mediaprovider.MediaProvider, track *mediaprovider.Track, count int) []*mediaprovider.Track {
	var tracks []*mediaprovider.Track
	if len(track.ArtistIDs) > 0 {
		tracks, _ = mp.GetSimilarTracks(track.ArtistIDs[0], count)
	}
	if len(tracks) == 0 {
		tracks, _ = mp.GetRandomTracks(track.Genre, count)
	}

	// make sure to exclude the song itself from the similar list
	return sharedutil.FilterSlice(tracks, func(t *mediaprovider.Track) bool {
		return t.ID != track.ID
	})
}
