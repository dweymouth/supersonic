package widgets

import (
	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

// Component that manages lazily loading more tracks into a Tracklist
// as the user scrolls near the bottom.
type TracklistLoader struct {
	disposed bool

	tracklist *Tracklist
	iter      mediaprovider.TrackIterator

	trackBuffer  []*mediaprovider.Track
	fetching     bool
	done         bool
	len          int
	highestShown int
}

func NewTracklistLoader(tracklist *Tracklist, iter mediaprovider.TrackIterator) TracklistLoader {
	t := TracklistLoader{
		tracklist: tracklist,
		iter:      iter,
	}
	t.tracklist.OnTrackShown = t.onTrackShown
	t.fetching = true
	go t.loadMoreTracks(25)
	return t
}

// Cancels all asynchronous loads so that they will no longer modify the tracklist.
func (t *TracklistLoader) Dispose() {
	t.disposed = true
	t.tracklist.OnTrackShown = nil
}

func (t *TracklistLoader) onTrackShown(tracknum int) {
	if tracknum > t.highestShown {
		t.highestShown = tracknum
	}
	if t.highestShown >= t.len-25 && !t.fetching && !t.done && !t.disposed {
		t.fetching = true
		go t.loadMoreTracks(25)
	}
}

func (t *TracklistLoader) loadMoreTracks(num int) {
	// repeat fetch task as long as user has scrolled near bottom
	for !t.done && t.highestShown >= t.len-25 {
		if t.trackBuffer == nil {
			t.trackBuffer = make([]*mediaprovider.Track, 0, num)
		}
		t.trackBuffer = t.trackBuffer[:0]
		for i := 0; i < num; i++ {
			tr := t.iter.Next()
			if tr == nil {
				t.done = true
				break
			}
			t.trackBuffer = append(t.trackBuffer, tr)
			if t.disposed {
				break
			}
		}
		if t.disposed {
			return
		}
		t.tracklist.AppendTracks(t.trackBuffer)
		t.tracklist.Refresh()
		t.len += len(t.trackBuffer)
	}
	if t.done {
		t.trackBuffer = nil
	}
	t.fetching = false
}
