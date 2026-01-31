package mpd

import (
	"context"
	"log"
	"strings"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/fhs/gompd/v2/mpd"
)

// Ensure mpdMediaProvider implements JukeboxWatcher
var _ mediaprovider.JukeboxWatcher = (*mpdMediaProvider)(nil)

// JukeboxProvider implementation for MPD
// MPD is always in "jukebox mode" - audio plays on the server

func (m *mpdMediaProvider) JukeboxStart() error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Play(-1) // -1 means continue from current position
	})
}

func (m *mpdMediaProvider) JukeboxPlay(idx int) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Play(idx)
	})
}

func (m *mpdMediaProvider) JukeboxStop() error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Pause(true)
	})
}

func (m *mpdMediaProvider) JukeboxSeek(idx, seconds int) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Seek(idx, seconds)
	})
}

func (m *mpdMediaProvider) JukeboxClear() error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Clear()
	})
}

func (m *mpdMediaProvider) JukeboxAdd(trackID string) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		// trackID is the file path in MPD
		return conn.Add(trackID)
	})
}

func (m *mpdMediaProvider) JukeboxRemove(idx int) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		return conn.Delete(idx, idx+1)
	})
}

func (m *mpdMediaProvider) JukeboxSet(trackID string) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		// Clear queue and add single track
		if err := conn.Clear(); err != nil {
			return err
		}
		return conn.Add(trackID)
	})
}

func (m *mpdMediaProvider) JukeboxSetVolume(vol int) error {
	return m.server.withConn(func(conn *mpd.Client) error {
		// MPD volume is 0-100
		if vol < 0 {
			vol = 0
		}
		if vol > 100 {
			vol = 100
		}
		return conn.SetVolume(vol)
	})
}

func (m *mpdMediaProvider) JukeboxGetStatus() (*mediaprovider.JukeboxStatus, error) {
	var status *mediaprovider.JukeboxStatus
	err := m.server.withConn(func(conn *mpd.Client) error {
		stat, err := conn.Status()
		if err != nil {
			return err
		}

		status = &mediaprovider.JukeboxStatus{
			Volume:          parseInt(stat["volume"]),
			CurrentTrack:    parseSongPosition(stat["song"]),
			Playing:         stat["state"] == "play",
			PositionSeconds: float64(parseSeconds(stat["elapsed"])) / float64(1e9), // Convert to seconds
		}

		// Parse bitrate (kbps)
		if bitrate := stat["bitrate"]; bitrate != "" {
			status.Bitrate = parseInt(bitrate)
		}

		// Parse audio format: "samplerate:bits:channels" (e.g., "44100:16:2")
		if audio := stat["audio"]; audio != "" {
			parts := strings.Split(audio, ":")
			if len(parts) >= 3 {
				status.SampleRate = parseInt(parts[0])
				status.BitDepth = parseInt(parts[1])
				status.Channels = parseInt(parts[2])
			}
		}

		// Get codec from current song info
		if currentSong, err := conn.CurrentSong(); err == nil {
			// MPD doesn't directly provide codec, but we can infer from file extension
			if file := currentSong["file"]; file != "" {
				if idx := strings.LastIndex(file, "."); idx != -1 {
					status.Codec = strings.ToLower(file[idx+1:])
				}
			}
		}

		return nil
	})
	return status, err
}

// JukeboxGetQueue returns the current MPD queue.
func (m *mpdMediaProvider) JukeboxGetQueue() ([]*mediaprovider.Track, int, error) {
	var tracks []*mediaprovider.Track
	var currentIdx int = -1

	err := m.server.withConn(func(conn *mpd.Client) error {
		// Get current status to know the playing position
		stat, err := conn.Status()
		if err != nil {
			return err
		}
		currentIdx = parseSongPosition(stat["song"])

		// Get the playlist info (queue)
		songs, err := conn.PlaylistInfo(-1, -1)
		if err != nil {
			return err
		}

		tracks = make([]*mediaprovider.Track, 0, len(songs))
		for _, song := range songs {
			if track := toTrack(song); track != nil {
				tracks = append(tracks, track)
			}
		}

		return nil
	})

	return tracks, currentIdx, err
}

// WatchPlaybackEvents uses MPD's idle command for efficient event-driven updates.
// This creates a separate connection since idle blocks until an event occurs.
func (m *mpdMediaProvider) WatchPlaybackEvents(ctx context.Context) (<-chan string, error) {
	// Create a new watcher connection
	watcher, err := mpd.NewWatcher("tcp", m.server.Hostname, m.server.password, "player", "mixer", "options")
	if err != nil {
		return nil, err
	}

	events := make(chan string, 10)

	go func() {
		defer watcher.Close()
		defer close(events)

		for {
			select {
			case <-ctx.Done():
				return
			case subsystem := <-watcher.Event:
				select {
				case events <- subsystem:
				default:
					// Channel full, skip event
				}
			case err := <-watcher.Error:
				if err != nil {
					log.Printf("MPD watcher error: %v", err)
				}
				// Watcher will attempt to reconnect automatically
			}
		}
	}()

	return events, nil
}
