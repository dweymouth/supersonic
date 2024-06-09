package ipc

const (
	PlayPath      = "/transport/play"
	PlayPausePath = "/transport/playpause"
	PausePath     = "/transport/pause"
	StopPath      = "/transport/stop"
	PreviousPath  = "/transport/previous"
	NextPath      = "/transport/next"
	TimePosPath   = "/transport/timepos"

	PlayTrackPath = "/queue/playtrack"

	VolumePath = "/volume"
)

type Response struct {
	Error string `json:"error"`
}
