package ipc

import "fmt"

const (
	PingPath             = "/ping"
	PlayPath             = "/transport/play"
	PlayPausePath        = "/transport/playpause"
	PausePath            = "/transport/pause"
	StopPath             = "/transport/stop"
	StopAfterCurrentPath = "/transport/stop-after-current"
	PreviousPath         = "/transport/previous"
	NextPath             = "/transport/next"
	TimePosPath          = "/transport/timepos" // ?s=<seconds>
	SeekByPath           = "/transport/seek-by" // ?s=<+/- seconds>
	VolumePath           = "/volume"            // ?v=<vol>
	VolumeAdjustPath     = "/volume/adjust"     // ?pct=<+/- percentage>
	ShowPath             = "/window/show"
	QuitPath             = "/window/quit"
)

type Response struct {
	Error string `json:"error"`
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
