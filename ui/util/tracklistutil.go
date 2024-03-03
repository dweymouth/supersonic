package util

import (
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
	from := minInt(idx, lastSelected)
	to := maxInt(idx, lastSelected)
	for i := from; i <= to; i++ {
		tracks[i].Selected = true
	}
}

func FindTrackByID(tracks []*TrackListModel, id string) (*mediaprovider.Track, int) {
	idx := sharedutil.Find(tracks, func(tr *TrackListModel) bool {
		return tr.Track.ID == id
	})
	if idx >= 0 {
		return tracks[idx].Track, idx
	}
	return nil, -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
