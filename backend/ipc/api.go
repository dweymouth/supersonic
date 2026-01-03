package ipc

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	PingPath              = "/ping"
	PlayPath              = "/transport/play"
	PlayAlbumPath         = "/transport/play-album"      // ?id=<album ID>&t=<firstTrack>&s=<shuffle>
	PlayPlaylistPath      = "/transport/play-playlist"   // ?id=<playlist ID>&t=<firstTrack>&s=<shuffle>
	PlayTrackPath         = "/transport/play-track"      // ?id=<track ID>
	SearchAlbumPath       = "/transport/search-album"    // ?s=<searchQuery>
	SearchPlaylistPath    = "/transport/search-playlist" // ?s=<searchQuery>
	SearchTrackPath       = "/transport/search-track"    // ?s=<searchQuery>
	PlayPausePath         = "/transport/playpause"
	PausePath             = "/transport/pause"
	StopPath              = "/transport/stop"
	PauseAfterCurrentPath = "/transport/pause-after-current"
	PreviousPath          = "/transport/previous"
	NextPath              = "/transport/next"
	TimePosPath           = "/transport/timepos" // ?s=<seconds>
	SeekByPath            = "/transport/seek-by" // ?s=<+/- seconds>
	VolumePath            = "/volume"            // ?v=<vol>
	VolumeAdjustPath      = "/volume/adjust"     // ?pct=<+/- percentage>
	ShowPath              = "/window/show"
	QuitPath              = "/window/quit"
)

type Response struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

func SetVolumePath(vol int) string {
	return fmt.Sprintf("%s?v=%d", VolumePath, vol)
}

func AdjustVolumePctPath(pct float64) string {
	return fmt.Sprintf("%s?pct=%0.2f", VolumeAdjustPath, pct)
}

func SeekToSecondsPath(secs float64) string {
	return fmt.Sprintf("%s?s=%0.2f", TimePosPath, secs)
}

func SeekBySecondsPath(secs float64) string {
	return fmt.Sprintf("%s?s=%0.2f", SeekByPath, secs)
}

func BuildPlayAlbumPath(id string, firstTrack int, shuffle bool) string {
	return fmt.Sprintf("%s?id=%s&t=%d&s=%t", PlayAlbumPath, id, firstTrack, shuffle)
}

func BuildPlayPlaylistPath(id string, firstTrack int, shuffle bool) string {
	return fmt.Sprintf("%s?id=%s&t=%d&s=%t", PlayPlaylistPath, id, firstTrack, shuffle)
}

func BuildPlayTrackPath(id string) string {
	return fmt.Sprintf("%s?id=%s", PlayTrackPath, id)
}

func BuildSearchAlbumPath(search string) string {
	s := url.QueryEscape(search)
	return fmt.Sprintf("%s?s=%s", SearchAlbumPath, s)
}

func BuildSearchPlaylistPath(search string) string {
	s := url.QueryEscape(search)
	return fmt.Sprintf("%s?s=%s", SearchPlaylistPath, s)
}

func BuildSearchTrackPath(search string) string {
	s := url.QueryEscape(search)
	return fmt.Sprintf("%s?s=%s", SearchTrackPath, s)
}
