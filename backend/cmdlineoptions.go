package backend

import (
	"flag"
	"strconv"
)

var (
	VolumeCLIArg int     = -1
	SeekToCLIArg float64 = -1

	FlagPlay      = flag.Bool("play", false, "unpause or begin playback")
	FlagPause     = flag.Bool("pause", false, "pause playback")
	FlagPlayPause = flag.Bool("play-pause", false, "toggle play/pause state")
	FlagPrevious  = flag.Bool("previous", false, "seek to previous track or beginning of current")
	FlagNext      = flag.Bool("next", false, "seek to next track")
	FlagVersion   = flag.Bool("version", false, "print app version and exit")
	FlagHelp      = flag.Bool("help", false, "print command line options and exit")
)

func init() {
	flag.Func("volume", "sets the playback volume (0-100)", func(s string) error {
		v, err := strconv.Atoi(s)
		VolumeCLIArg = v
		return err
	})

	flag.Func("seek-to", "seeks to the given position in seconds in the current file (0.0 - <trackDur>)", func(s string) error {
		v, err := strconv.ParseFloat(s, 64)
		SeekToCLIArg = v
		return err
	})
}

func HaveCommandLineOptions() bool {
	visitedAny := false
	flag.Visit(func(*flag.Flag) {
		visitedAny = true
	})
	return visitedAny
}
