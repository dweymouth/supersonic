package sharedutil

import (
	"slices"
	"testing"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

func Test_ReorderTracks(t *testing.T) {
	tracks := []*mediaprovider.Track{
		{ID: "a"}, // 0
		{ID: "b"}, // 1
		{ID: "c"}, // 2
		{ID: "d"}, // 3
		{ID: "e"}, // 4
		{ID: "f"}, // 5
	}

	// test MoveToTop:
	idxToMove := []int{0, 2, 3, 5}
	want := []*mediaprovider.Track{
		{ID: "a"},
		{ID: "c"},
		{ID: "d"},
		{ID: "f"},
		{ID: "b"},
		{ID: "e"},
	}
	newTracks := ReorderTracks(tracks, idxToMove, MoveToTop)
	if !tracklistsEqual(t, newTracks, want) {
		t.Error("ReorderTracks: MoveToTop order incorrect")
	}

	// test MoveToBottom:
	idxToMove = []int{0, 2, 5}
	want = []*mediaprovider.Track{
		{ID: "b"},
		{ID: "d"},
		{ID: "e"},
		{ID: "a"},
		{ID: "c"},
		{ID: "f"},
	}
	newTracks = ReorderTracks(tracks, idxToMove, MoveToBottom)
	if !tracklistsEqual(t, newTracks, want) {
		t.Error("ReorderTracks: MoveToBottom order incorrect")
	}

	// test MoveUp:
	idxToMove = []int{0, 1, 3, 5}
	want = []*mediaprovider.Track{
		{ID: "a"},
		{ID: "b"},
		{ID: "d"},
		{ID: "c"},
		{ID: "f"},
		{ID: "e"},
	}
	newTracks = ReorderTracks(tracks, idxToMove, MoveUp)
	if !tracklistsEqual(t, newTracks, want) {
		t.Error("ReorderTracks: MoveUp order incorrect")
	}

	// test MoveDown:
	idxToMove = []int{2, 4, 5}
	want = []*mediaprovider.Track{
		{ID: "a"},
		{ID: "b"},
		{ID: "d"},
		{ID: "c"},
		{ID: "e"},
		{ID: "f"},
	}
	newTracks = ReorderTracks(tracks, idxToMove, MoveDown)
	if !tracklistsEqual(t, newTracks, want) {
		t.Error("ReorderTracks: MoveDown order incorrect")
	}
}

func tracklistsEqual(t *testing.T, a, b []*mediaprovider.Track) bool {
	t.Helper()
	return slices.EqualFunc(a, b, func(a, b *mediaprovider.Track) bool {
		return a.ID == b.ID
	})
}
