package ipc

const (
	// GET
	PingPath = "/ping"
	// POST
	PlayPath = "/transport/play"
	// POST
	PlayPausePath = "/transport/playpause"
	// POST
	PausePath = "/transport/pause"
	// POST
	StopPath = "/transport/stop"
	// POST
	PreviousPath = "/transport/previous"
	// POST
	NextPath = "/transport/next"
	// POST(TimePos)
	TimePosPath = "/transport/timepos"
	// POST to seek
	PlayTrackPath = "/queue/playtrack"
	// GET -> Volume
	// POST(Volume)
	VolumePath = "/volume"
)

type TimePos struct {
	Seconds float64 `json:"seconds"`
}

type Volume struct {
	Volume int `json:"volume"`
}

type Response struct {
	Error string `json:"error"`
}
