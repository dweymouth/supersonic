package backend

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
)

type SavedPlayQueue struct {
	Tracks     []*mediaprovider.Track
	TrackIndex int
	TimePos    float64
}

type serializedSavedPlayQueue struct {
	ServerID   string   `json:"serverID"`
	TrackIDs   []string `json:"trackIDs"`
	TrackIndex int      `json:"trackIndex"`
	TimePos    float64  `json:"timePos"`
}

// SavePlayQueue saves the current play queue and playback position to a JSON file.
func SavePlayQueue(serverID string, pm *PlaybackManager, filepath string) error {
	queue := pm.GetPlayQueue()
	stats := pm.PlayerStatus()
	trackIdx := pm.NowPlayingIndex()

	trackIDs := make([]string, len(queue))
	for i, tr := range queue {
		trackIDs[i] = tr.ID
	}

	saved := serializedSavedPlayQueue{
		ServerID:   serverID,
		TrackIDs:   trackIDs,
		TrackIndex: trackIdx,
		TimePos:    stats.TimePos,
	}
	b, err := json.Marshal(saved)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, b, 0644)
}

// Loads the saved play queue from the given filepath using the current server.
// Returns an error if the queue could not be loaded for any reason, including the
// currently logged in server being different than the server from which the queue was saved.
func LoadPlayQueue(filepath string, sm *ServerManager) (*SavedPlayQueue, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var savedData serializedSavedPlayQueue
	if err := json.Unmarshal(b, &savedData); err != nil {
		return nil, err
	}

	if sm.ServerID.String() != savedData.ServerID {
		return nil, errors.New("saved play queue was from a different server")
	}

	tracks := make([]*mediaprovider.Track, 0, len(savedData.TrackIDs))
	mp := sm.Server
	for i, id := range savedData.TrackIDs {
		if tr, err := mp.GetTrack(id); err != nil {
			// ignore/skip individual track failures
			if i < savedData.TrackIndex {
				savedData.TrackIndex--
			}
		} else {
			tracks = append(tracks, tr)
		}
	}

	savedQueue := &SavedPlayQueue{
		Tracks:     tracks,
		TrackIndex: savedData.TrackIndex,
		TimePos:    savedData.TimePos,
	}
	return savedQueue, nil
}
