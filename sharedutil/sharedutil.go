package sharedutil

import "github.com/dweymouth/go-subsonic/subsonic"

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
