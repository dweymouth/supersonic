package sharedutil

import "github.com/dweymouth/go-subsonic/subsonic"

func StringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
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
