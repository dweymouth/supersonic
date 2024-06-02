package sharedutil

import (
	"math"
	"slices"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

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

func FilterMapSlice[T any, U any](ts []T, f func(T) (U, bool)) []U {
	if ts == nil {
		return nil
	}
	result := make([]U, 0)
	for _, t := range ts {
		if u, ok := f(t); ok {
			result = append(result, u)
		}
	}
	return result
}

func Reversed[T any](ts []T) []T {
	if ts == nil {
		return nil
	}
	new := make([]T, len(ts))
	j := len(ts) - 1
	for i := range ts {
		new[i] = ts[j]
		j--
	}
	return new
}

func ToSet[T comparable](ts []T) map[T]struct{} {
	set := make(map[T]struct{}, len(ts))
	for _, t := range ts {
		set[t] = struct{}{}
	}
	return set
}

func FindTrackByID(id string, tracks []*mediaprovider.Track) *mediaprovider.Track {
	for _, tr := range tracks {
		if id == tr.ID {
			return tr
		}
	}
	return nil
}

func FindMediaItemByID(id string, items []mediaprovider.MediaItem) mediaprovider.MediaItem {
	for _, tr := range items {
		if id == tr.Metadata().ID {
			return tr
		}
	}
	return nil
}

func MediaItemIDOrEmptyStr(item mediaprovider.MediaItem) string {
	if tr, ok := item.(*mediaprovider.Track); ok && tr != nil {
		return tr.ID
	}
	if rd, ok := item.(*mediaprovider.RadioStation); ok && rd != nil {
		return rd.ID
	}
	return ""
}

func AlbumIDOrEmptyStr(track *mediaprovider.Track) string {
	if track == nil {
		return ""
	}
	return track.AlbumID
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

// Reorder items and return a new track slice.
// idxToMove must contain only valid indexes into tracks, and no repeats
func ReorderItems[T any](items []T, idxToMove []int, op TrackReorderOp) []T {
	newItems := make([]T, len(items))
	switch op {
	case MoveToTop:
		topIdx := 0
		botIdx := len(idxToMove)
		idxToMoveSet := ToSet(idxToMove)
		for i, t := range items {
			if _, ok := idxToMoveSet[i]; ok {
				newItems[topIdx] = t
				topIdx++
			} else {
				newItems[botIdx] = t
				botIdx++
			}
		}
	case MoveToBottom:
		topIdx := 0
		botIdx := len(items) - len(idxToMove)
		idxToMoveSet := ToSet(idxToMove)
		for i, t := range items {
			if _, ok := idxToMoveSet[i]; ok {
				newItems[botIdx] = t
				botIdx++
			} else {
				newItems[topIdx] = t
				topIdx++
			}
		}
	case MoveUp:
		first := firstIdxCanMoveUp(idxToMove)
		copy(newItems, items)
		for _, i := range idxToMove {
			if i < first {
				continue
			}
			newItems[i-1], newItems[i] = newItems[i], newItems[i-1]
		}
	case MoveDown:
		last := lastIdxCanMoveDown(idxToMove, len(items))
		copy(newItems, items)
		for i := len(idxToMove) - 1; i >= 0; i-- {
			idx := idxToMove[i]
			if idx > last {
				continue
			}
			newItems[idx+1], newItems[idx] = newItems[idx], newItems[idx+1]
		}
	}
	return newItems
}

func firstIdxCanMoveUp(idxs []int) int {
	prevIdx := -1
	slices.Sort(idxs)
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
	slices.Sort(idxs)
	for i := len(idxs) - 1; i >= 0; i-- {
		idx := idxs[i]
		if idx < prevIdx-1 {
			return idx
		}
		prevIdx = idx
	}
	return -1
}
