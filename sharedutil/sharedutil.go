package sharedutil

import (
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

// Reorder items and return a new track slice.
// idxToMove must contain only valid indexes into tracks, and no repeats
func ReorderItems[T any](items []T, idxToMove []int, insertIdx int) []T {
	idxToMoveSet := ToSet(idxToMove)

	newItems := make([]T, 0, len(items))

	// collect items that will end up before the insertion set
	i := 0
	for ; i < len(items); i++ {
		if insertIdx == i {
			break
		}
		if _, ok := idxToMoveSet[i]; !ok {
			newItems = append(newItems, items[i])
		}
	}

	for _, idx := range idxToMove {
		newItems = append(newItems, items[idx])
	}

	for ; i < len(items); i++ {
		if _, ok := idxToMoveSet[i]; !ok {
			newItems = append(newItems, items[i])
		}
	}

	return newItems
}
