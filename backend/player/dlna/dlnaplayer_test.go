package dlna

import (
	"testing"
	"time"
)

// TestDLNATimingConstants verifies that DLNA timing constants are set to reasonable values
func TestDLNATimingConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "maxSeekRetries",
			value:    maxSeekRetries,
			expected: 5,
		},
		{
			name:     "seekRetryInitialDelay",
			value:    seekRetryInitialDelay,
			expected: 400 * time.Millisecond,
		},
		{
			name:     "seekRetryMaxDelay",
			value:    seekRetryMaxDelay,
			expected: 2 * time.Second,
		},
		{
			name:     "playbackSyncDelay",
			value:    playbackSyncDelay,
			expected: 500 * time.Millisecond,
		},
		{
			name:     "playbackSyncRetries",
			value:    playbackSyncRetries,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, expected %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// TestExponentialBackoff verifies the exponential backoff calculation behavior
func TestExponentialBackoff(t *testing.T) {
	// Simulate the exponential backoff logic from sendSeekCmd
	delay := seekRetryInitialDelay
	expectedDelays := []time.Duration{
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
		2000 * time.Millisecond, // capped at seekRetryMaxDelay
		2000 * time.Millisecond, // stays capped
	}

	for i := 0; i < maxSeekRetries; i++ {
		if delay != expectedDelays[i] {
			t.Errorf("Attempt %d: delay = %v, expected %v", i+1, delay, expectedDelays[i])
		}

		// Simulate the backoff logic
		delay *= 2
		if delay > seekRetryMaxDelay {
			delay = seekRetryMaxDelay
		}
	}
}

// TestFormatTime verifies DLNA time format conversion
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{
			name:     "zero seconds",
			seconds:  0,
			expected: "00:00:00",
		},
		{
			name:     "one minute",
			seconds:  60,
			expected: "00:01:00",
		},
		{
			name:     "one hour",
			seconds:  3600,
			expected: "01:00:00",
		},
		{
			name:     "complex time",
			seconds:  3661,
			expected: "01:01:01",
		},
		{
			name:     "typical song duration",
			seconds:  195, // 3:15
			expected: "00:03:15",
		},
		{
			name:     "long track",
			seconds:  7384, // 2:03:04
			expected: "02:03:04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatTime(%d) = %s, expected %s", tt.seconds, result, tt.expected)
			}
		})
	}
}

// TestBuildDIDLMetadata verifies DIDL-Lite XML generation
func TestBuildDIDLMetadata(t *testing.T) {
	tests := []struct {
		name     string
		media    *MediaItem
		wantXML  bool
		contains []string
	}{
		{
			name:    "nil media",
			media:   nil,
			wantXML: false,
		},
		{
			name: "empty URL",
			media: &MediaItem{
				URL:         "",
				Title:       "Test",
				ContentType: "audio/mp3",
			},
			wantXML: false,
		},
		{
			name: "valid media item",
			media: &MediaItem{
				URL:         "http://example.com/song.mp3",
				Title:       "Test Song",
				ContentType: "audio/mpeg",
				Seekable:    true,
			},
			wantXML: true,
			contains: []string{
				"<DIDL-Lite",
				"<dc:title>Test Song</dc:title>",
				"http://example.com/song.mp3",
				"audio/mpeg",
				"object.item.audioItem.musicTrack",
			},
		},
		{
			name: "special characters in title",
			media: &MediaItem{
				URL:         "http://example.com/track.flac",
				Title:       "Song & Title",
				ContentType: "audio/flac",
				Seekable:    true,
			},
			wantXML: true,
			contains: []string{
				"<dc:title>Song & Title</dc:title>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDIDLMetadata(tt.media)

			if !tt.wantXML {
				if result != "" {
					t.Errorf("buildDIDLMetadata() should return empty string, got %s", result)
				}
				return
			}

			for _, substr := range tt.contains {
				if !contains(result, substr) {
					t.Errorf("buildDIDLMetadata() missing expected substring: %s", substr)
				}
			}
		})
	}
}

// TestMediaItem verifies MediaItem struct
func TestMediaItem(t *testing.T) {
	media := MediaItem{
		URL:         "http://test.com/file.mp3",
		Title:       "Test Track",
		ContentType: "audio/mpeg",
		Seekable:    true,
	}

	if media.URL != "http://test.com/file.mp3" {
		t.Errorf("URL = %s, expected http://test.com/file.mp3", media.URL)
	}
	if media.Title != "Test Track" {
		t.Errorf("Title = %s, expected Test Track", media.Title)
	}
	if media.ContentType != "audio/mpeg" {
		t.Errorf("ContentType = %s, expected audio/mpeg", media.ContentType)
	}
	if !media.Seekable {
		t.Error("Seekable should be true")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && containsLoop(s, substr))
}

func containsLoop(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
