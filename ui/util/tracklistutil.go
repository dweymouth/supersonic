package util

import (
	"slices"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/dweymouth/supersonic/sharedutil"
)

type TrackListModel struct {
	Track    *mediaprovider.Track
	Selected bool
}

func ToTrackListModels(trs []*mediaprovider.Track) []*TrackListModel {
	return sharedutil.MapSlice(trs, func(tr *mediaprovider.Track) *TrackListModel {
		return &TrackListModel{Track: tr, Selected: false}
	})
}

func SelectedTrackModels(tracks []*TrackListModel) []*TrackListModel {
	return sharedutil.FilterSlice(tracks, func(tm *TrackListModel) bool {
		return tm.Selected
	})
}

func SelectedTracks(tracks []*TrackListModel) []*mediaprovider.Track {
	return sharedutil.MapSlice(SelectedTrackModels(tracks), func(tm *TrackListModel) *mediaprovider.Track {
		return tm.Track
	})
}

func SelectedTrackIDs(tracks []*TrackListModel) []string {
	return sharedutil.MapSlice(SelectedTrackModels(tracks), func(tm *TrackListModel) string {
		return tm.Track.ID
	})
}

func SelectTrack(tracks []*TrackListModel, idx int) {
	if tracks[idx].Selected {
		return
	}
	UnselectAllTracks(tracks)
	tracks[idx].Selected = true
}

func SelectAllTracks(tracks []*TrackListModel) {
	for _, tm := range tracks {
		tm.Selected = true
	}
}

func UnselectAllTracks(tracks []*TrackListModel) {
	for _, tm := range tracks {
		tm.Selected = false
	}
}

func SelectTrackRange(tracks []*TrackListModel, idx int) {
	if tracks[idx].Selected {
		return
	}
	lastSelected := -1
	for i := len(tracks) - 1; i >= 0; i-- {
		if tracks[i].Selected {
			lastSelected = i
			break
		}
	}
	if lastSelected < 0 {
		tracks[idx].Selected = true
		return
	}
	from := min(idx, lastSelected)
	to := max(idx, lastSelected)
	for i := from; i <= to; i++ {
		tracks[i].Selected = true
	}
}

func FindTrackByID(tracks []*TrackListModel, id string) (*mediaprovider.Track, int) {
	idx := slices.IndexFunc(tracks, func(tr *TrackListModel) bool {
		return tr.Track.ID == id
	})
	if idx >= 0 {
		return tracks[idx].Track, idx
	}
	return nil, -1
}
