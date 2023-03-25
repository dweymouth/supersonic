package sharedutil

import (
	"testing"

	"github.com/dweymouth/go-subsonic/subsonic"
)

func Test_ReorderTracks(t *testing.T) {
	tracks := []*subsonic.Child{
		{ID: "a"}, // 0
		{ID: "b"}, // 1
		{ID: "c"}, // 2
		{ID: "d"}, // 3
		{ID: "e"}, // 4
		{ID: "f"}, // 5
	}

	// test MoveToTop:
	idxToMove := []int{0, 2, 3, 5}
	want := []*subsonic.Child{
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
	want = []*subsonic.Child{
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
	want = []*subsonic.Child{
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
	want = []*subsonic.Child{
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

func tracklistsEqual(t *testing.T, a, b []*subsonic.Child) bool {
	t.Helper()
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}
