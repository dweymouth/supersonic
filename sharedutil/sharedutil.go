package sharedutil

import (
	"math"
	"sort"

	"github.com/dweymouth/go-subsonic/subsonic"
)

func StringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func IntSliceContains(slice []int, i int) bool {
	for _, x := range slice {
		if x == i {
			return true
		}
	}
	return false
}

func FindTrackByID(id string, tracks []*subsonic.Child) *subsonic.Child {
	for _, tr := range tracks {
		if id == tr.ID {
			return tr
		}
	}
	return nil
}

func TrackIDOrEmptyStr(track *subsonic.Child) string {
	if track == nil {
		return ""
	}
	return track.ID
}

type TrackReorderOp int

const (
	MoveToTop TrackReorderOp = iota
	MoveToBottom
	MoveUp
	MoveDown
)

// Reorder tracks and return a new track slice.
// idxToMove must contain only valid indexes into tracks, and no repeats
func ReorderTracks(tracks []*subsonic.Child, idxToMove []int, op TrackReorderOp) []*subsonic.Child {
	newTracks := make([]*subsonic.Child, len(tracks))
	switch op {
	case MoveToTop:
		topIdx := 0
		botIdx := len(idxToMove)
		for i, t := range tracks {
			if IntSliceContains(idxToMove, i) {
				newTracks[topIdx] = t
				topIdx++
			} else {
				newTracks[botIdx] = t
				botIdx++
			}
		}
	case MoveToBottom:
		topIdx := 0
		botIdx := len(tracks) - len(idxToMove)
		for i, t := range tracks {
			if IntSliceContains(idxToMove, i) {
				newTracks[botIdx] = t
				botIdx++
			} else {
				newTracks[topIdx] = t
				topIdx++
			}
		}
	case MoveUp:
		first := firstIdxCanMoveUp(idxToMove)
		copy(newTracks, tracks)
		for _, i := range idxToMove {
			if i < first {
				continue
			}
			newTracks[i-1], newTracks[i] = newTracks[i], newTracks[i-1]
		}
	case MoveDown:
		last := lastIdxCanMoveDown(idxToMove, len(tracks))
		copy(newTracks, tracks)
		for i := len(idxToMove) - 1; i >= 0; i-- {
			idx := idxToMove[i]
			if idx > last {
				continue
			}
			newTracks[idx+1], newTracks[idx] = newTracks[idx], newTracks[idx+1]
		}
	}
	return newTracks
}

func firstIdxCanMoveUp(idxs []int) int {
	prevIdx := -1
	sort.Ints(idxs)
	for _, idx := range idxs {
		if idx > prevIdx+1 {
			return idx
		}
		prevIdx = idx
	}
	return math.MaxInt
}

func lastIdxCanMoveDown(idxs []int, lenSlice int) int {
	prevIdx := lenSlice
	sort.Ints(idxs)
	for i := len(idxs) - 1; i >= 0; i-- {
		idx := idxs[i]
		if idx < prevIdx-1 {
			return idx
		}
		prevIdx = idx
	}
	return -1
}
