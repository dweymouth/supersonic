package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

// FetchLrcLibLyrics is a static function to search and fetch lyrics from lrclib.net
func FetchLrcLibLyrics(name, artist, album string, durationSecs int) (*mediaprovider.Lyrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://lrclib.net/api/get", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "Supersonic")

	q := req.URL.Query()
	q.Add("track_name", name)
	q.Add("artist_name", artist)
	q.Add("album_name", album)
	q.Add("duration", strconv.Itoa(durationSecs))
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseLrcLibResponse(resp)
}

func parseLrcLibResponse(resp *http.Response) (*mediaprovider.Lyrics, error) {
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("lrclib lyrics not found")
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from lrclib: status %d", resp.StatusCode)
	}

	var parsedResponse lrcLibResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsedResponse); err != nil {
		return nil, fmt.Errorf("failed to decode lrclib response: %w", err)
	}
	lrcs := &mediaprovider.Lyrics{
		Title:  parsedResponse.TrackName,
		Artist: parsedResponse.ArtistName,
	}
	if parsedResponse.SyncedLyrics != "" {
		lines, err := parseSyncedLyrics(parsedResponse.SyncedLyrics)
		if err != nil {
			return nil, err
		}
		lrcs.Synced = true
		lrcs.Lines = lines
	} else {
		for _, line := range strings.Split(parsedResponse.PlainLyrics, "\n") {
			lrcs.Lines = append(lrcs.Lines, mediaprovider.LyricLine{Text: line})
		}
	}
	return lrcs, nil
}

var syncedRegex = regexp.MustCompile(`^\[(\d\d):(\d\d\.\d\d\d?)\] ?(.+)$`)

func parseSyncedLyrics(synced string) ([]mediaprovider.LyricLine, error) {
	var lines []mediaprovider.LyricLine
	for _, line := range strings.Split(synced, "\n") {
		matches := syncedRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue // malformed lyric line, attempt to continue
		}
		min, _ := strconv.Atoi(matches[1])
		sec, _ := strconv.ParseFloat(matches[2], 64)
		lines = append(lines, mediaprovider.LyricLine{
			Start: float64(min)*60 + sec,
			Text:  matches[3],
		})
	}
	var err error
	if len(lines) == 0 {
		err = errors.New("failed to parse synced lyrics")
	}
	return lines, err
}

type lrcLibResponse struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}
