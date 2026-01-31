package mpd

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dweymouth/supersonic/backend/mediaprovider"
	"github.com/fhs/gompd/v2/mpd"
)

var (
	ErrNotConnected = errors.New("not connected to MPD")
	ErrNotSupported = errors.New("operation not supported by MPD")
)

// MPDServer implements mediaprovider.Server for MPD connections.
type MPDServer struct {
	Hostname string
	Language string // User's preferred language for Wikipedia biographies

	mu       sync.RWMutex
	conn     *mpd.Client
	password string
	provider *mpdMediaProvider
}

// Login connects to the MPD server. The username is ignored (MPD doesn't use usernames).
// The password may be empty if the MPD server doesn't require authentication.
func (s *MPDServer) Login(username, password string) mediaprovider.LoginResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, err := mpd.Dial("tcp", s.Hostname)
	if err != nil {
		return mediaprovider.LoginResponse{
			Error:       fmt.Errorf("failed to connect to MPD: %w", err),
			IsAuthError: false,
		}
	}

	// If a password is provided, authenticate
	if password != "" {
		if err := conn.Command("password %s", password).OK(); err != nil {
			conn.Close()
			return mediaprovider.LoginResponse{
				Error:       fmt.Errorf("MPD authentication failed: %w", err),
				IsAuthError: true,
			}
		}
	}

	// Connection is established - the Dial already verified connectivity
	s.conn = conn
	s.password = password
	s.provider = &mpdMediaProvider{
		server:            s,
		artistInfoFetcher: newArtistInfoFetcher(s.Language),
	}

	return mediaprovider.LoginResponse{}
}

// MediaProvider returns the MediaProvider implementation.
func (s *MPDServer) MediaProvider() mediaprovider.MediaProvider {
	return s.provider
}

// reconnect attempts to reconnect to the MPD server.
func (s *MPDServer) reconnect(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	conn, err := mpd.Dial("tcp", s.Hostname)
	if err != nil {
		return fmt.Errorf("failed to reconnect to MPD: %w", err)
	}

	if password != "" {
		if err := conn.Command("password %s", password).OK(); err != nil {
			conn.Close()
			return fmt.Errorf("MPD authentication failed during reconnect: %w", err)
		}
	}

	s.conn = conn
	return nil
}

// getConn returns the MPD connection.
// Connection health is checked via error handling in withConn, not by pinging on every call.
func (s *MPDServer) getConn() (*mpd.Client, error) {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return nil, ErrNotConnected
	}

	return conn, nil
}

// withConn executes a function with the MPD connection, handling reconnection if needed.
func (s *MPDServer) withConn(fn func(*mpd.Client) error) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}

	// Execute the function
	err = fn(conn)
	if err != nil {
		// Check if it's a connection error
		if isConnectionError(err) {
			// Try to reconnect and retry once (using stored password)
			if reconnErr := s.reconnect(s.password); reconnErr != nil {
				return reconnErr
			}
			conn, err = s.getConn()
			if err != nil {
				return err
			}
			return fn(conn)
		}
	}
	return err
}

// isConnectionError checks if an error indicates a lost connection.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "closed") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "broken pipe")
}

// parseSeconds parses a duration string (e.g., "123.456") into time.Duration.
func parseSeconds(s string) time.Duration {
	var seconds float64
	fmt.Sscanf(s, "%f", &seconds)
	return time.Duration(seconds * float64(time.Second))
}

// parseInt parses a string to an integer, returning 0 on error or empty string.
func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// parseSongPosition parses MPD's song position field, returning -1 for empty/missing values.
// This is used for the "song" status field which is empty when no song is playing.
func parseSongPosition(s string) int {
	if s == "" {
		return -1
	}
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
