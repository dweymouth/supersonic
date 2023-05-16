package sharedutil

import (
	"math"
	"sort"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

func SliceContains[T comparable](ts []T, t T) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

func FilterSlice[T any](ss []T, test func(T) bool) []T {
	if ss == nil {
		return nil
	}
	result := make([]T, 0)
	for _, s := range ss {
		if test(s) {
			result = append(result, s)
		}
	}
	return result
}

func MapSlice[T any, U any](ts []T, f func(T) U) []U {
	if ts == nil {
		return nil
	}
	result := make([]U, len(ts))
	for i, t := range ts {
		result[i] = f(t)
	}
	return result
}

func FindTrackByID(id string, tracks []*mediaprovider.Track) *mediaprovider.Track {
	for _, tr := range tracks {
		if id == tr.ID {
			return tr
		}
	}
	return nil
}

func TrackIDOrEmptyStr(track *mediaprovider.Track) string {
	if track == nil {
		return ""
	}
	return track.ID
}

func TracksToIDs(tracks []*mediaprovider.Track) []string {
	return MapSlice(tracks, func(tr *mediaprovider.Track) string {
		return tr.ID
	})
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
func ReorderTracks(tracks []*mediaprovider.Track, idxToMove []int, op TrackReorderOp) []*mediaprovider.Track {
	newTracks := make([]*mediaprovider.Track, len(tracks))
	switch op {
	case MoveToTop:
		topIdx := 0
		botIdx := len(idxToMove)
		for i, t := range tracks {
			if SliceContains(idxToMove, i) {
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
			if SliceContains(idxToMove, i) {
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
