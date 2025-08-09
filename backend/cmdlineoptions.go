package backend

import (
	"flag"
	"strconv"
	"strings"
)

var (
	VolumeCLIArg       int     = -1
	SeekToCLIArg       float64 = -1
	SeekByCLIArg       float64 = 0
	VolumePctCLIArg    float64 = 0
	PlayAlbumCLIArg    string  = ""
	PlayPlaylistCLIArg string  = ""
	PlayTrackCLIArg    string  = ""
	FirstTrackCLIArg   int     = 0

	FlagPlay           = flag.Bool("play", false, "unpause or begin playback")
	FlagPause          = flag.Bool("pause", false, "pause playback")
	FlagPlayPause      = flag.Bool("play-pause", false, "toggle play/pause state")
	FlagPrevious       = flag.Bool("previous", false, "seek to previous track or beginning of current")
	FlagNext           = flag.Bool("next", false, "seek to next track")
	FlagStartMinimized = flag.Bool("start-minimized", false, "start app minimized")
	FlagShow           = flag.Bool("show", false, "show minimized app")
	FlagShuffle        = flag.Bool("shuffle", false, "shuffle the tracklist (to be used with either -play-album or -play-playlist)")
	FlagVersion        = flag.Bool("version", false, "print app version and exit")
	FlagHelp           = flag.Bool("help", false, "print command line options and exit")
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
	flag.Func("seek-by", "seeks back or forward by the given number of seconds (negative or positive)", func(s string) error {
		v, err := strconv.ParseFloat(s, 64)
		SeekByCLIArg = v
		return err
	})
	flag.Func("volume-adjust-pct", "adjusts volume up or down by the given percentage (positive or negative)", func(s string) error {
		if strings.HasSuffix(s, "%") {
			s = s[:len(s)-1]
		}
		v, err := strconv.ParseFloat(s, 64)
		VolumePctCLIArg = v
		return err
	})

	flag.Func("play-album-by-id", "start playing the given album (ID)", func(s string) error {
		PlayAlbumCLIArg = s
		return nil
	})
	flag.Func("play-playlist-by-id", "start playing the given playlist (ID)", func(s string) error {
		PlayPlaylistCLIArg = s
		return nil
	})
	flag.Func("play-track-by-id", "start playing the given track (ID)", func(s string) error {
		PlayTrackCLIArg = s
		return nil
	})
	flag.Func("first-track", "start playing from given track (positive integer, to be used with either -play-album or -play-playlist)", func(s string) error {
		v, err := strconv.Atoi(s)
		FirstTrackCLIArg = v
		return err
	})
}

func HaveCommandLineOptions() bool {
	visitedAny := false
	flag.Visit(func(f *flag.Flag) {
		// We skip `start-minimized` because it should't send an IPC message.
		if f.Name != "start-minimized" {
			visitedAny = true
		}
	})
	return visitedAny
}
