package backend

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var (
	VolumeCLIArg         int     = -1
	SeekToCLIArg         float64 = -1
	SeekByCLIArg         float64 = 0
	VolumePctCLIArg      float64 = 0
	PlayAlbumCLIArg      string  = ""
	PlayPlaylistCLIArg   string  = ""
	PlayTrackCLIArg      string  = ""
	FirstTrackCLIArg     int     = 0
	SearchAlbumCLIArg    string  = ""
	SearchPlaylistCLIArg string  = ""
	SearchTrackCLIArg    string  = ""

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

	FlagPlayAlbum    *bool
	FlagPlayPlaylist *bool
	FlagPlayTrack    *bool
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

	if term.IsTerminal(int(os.Stdin.Fd())) {
		flag.Func("play-album-by-id", "start playing the album with the given ID (can also be passed from standard input)", func(s string) error {
			PlayAlbumCLIArg = s
			return nil
		})
		flag.Func("play-playlist-by-id", "start playing the playlist with the given ID (can also be passed from standard input)", func(s string) error {
			PlayPlaylistCLIArg = s
			return nil
		})
		flag.Func("play-track-by-id", "start playing the track with the given ID (can also be passed from standard input)", func(s string) error {
			PlayTrackCLIArg = s
			return nil
		})
	} else {
		FlagPlayAlbum = flag.Bool("play-album-by-id", false, "")
		FlagPlayPlaylist = flag.Bool("play-playlist-by-id", false, "")
		FlagPlayTrack = flag.Bool("play-track-by-id", false, "")
	}
	flag.Func("first-track", "start playing from given track (positive integer, to be used with either -play-album or -play-playlist)", func(s string) error {
		v, err := strconv.Atoi(s)
		FirstTrackCLIArg = v
		return err
	})

	flag.Func("search-album", "search album", func(s string) error {
		SearchAlbumCLIArg = s
		return nil
	})
	flag.Func("search-playlist", "search playlist", func(s string) error {
		SearchPlaylistCLIArg = s
		return nil
	})
	flag.Func("search-track", "search track", func(s string) error {
		SearchTrackCLIArg = s
		return nil
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
