package widgets

import (
	"log"
	"supersonic/backend"

	"github.com/dweymouth/go-subsonic/subsonic"
)

// Component that manages lazily loading more tracks into a Tracklist
// as the user scrolls near the bottom.
type TracklistLoader struct {
	tracklist *Tracklist
	iter      backend.TrackIterator

	trackBuffer  []*subsonic.Child
	fetching     bool
	done         bool
	len          int
	highestShown int
}

func NewTracklistLoader(tracklist *Tracklist, iter backend.TrackIterator) TracklistLoader {
	t := TracklistLoader{
		tracklist: tracklist,
		iter:      iter,
	}
	t.tracklist.OnTrackShown = t.onTrackShown
	t.fetching = true
	go t.loadMoreTracks(25)
	return t
}

func (t *TracklistLoader) onTrackShown(tracknum int) {
	if tracknum > t.highestShown {
		t.highestShown = tracknum
	}
	if t.highestShown >= t.len-25 && !t.fetching && !t.done {
		t.fetching = true
		go t.loadMoreTracks(25)
	}
}

func (t *TracklistLoader) loadMoreTracks(num int) {
	// repeat fetch task as long as user has scrolled near bottom
	for !t.done && t.highestShown >= t.len-25 {
		log.Println("fetching more tracks")
		if t.trackBuffer == nil {
			t.trackBuffer = make([]*subsonic.Child, 0, num)
		}
		t.trackBuffer = t.trackBuffer[:0]
		for i := 0; i < num; i++ {
			tr := t.iter.Next()
			if tr == nil {
				t.done = true
				t.trackBuffer = nil
				break
			}
			t.trackBuffer = append(t.trackBuffer, tr)
		}
		t.tracklist.AppendTracks(t.trackBuffer)
		t.tracklist.Refresh()
		log.Printf("appended %d tracks", len(t.trackBuffer))
		t.len += len(t.trackBuffer)
	}
	t.fetching = false
}
