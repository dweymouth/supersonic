package mpd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
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

// Compile-time interface assertions for MPDServer
var _ mediaprovider.JukeboxOnlyServer = (*MPDServer)(nil)
var _ mediaprovider.JukeboxProvider = (*MPDServer)(nil)
var _ mediaprovider.CacheManager = (*MPDServer)(nil)

const (
	maxPoolSize = 5 // Maximum number of concurrent connections (MPD default limit is often 10)
)

// MPDServer implements mediaprovider.Server for MPD connections.
type MPDServer struct {
	Hostname string
	Language string // User's preferred language for Wikipedia biographies

	mu          sync.Mutex // Protects the pool and connection state
	pool        []*mpd.Client
	password    string
	provider    *mpdMediaProvider
	connected   bool
	activeConns int           // Number of connections currently in use
	connSem     chan struct{} // Semaphore to limit concurrent connections
}

// Login connects to the MPD server. The username is ignored (MPD doesn't use usernames).
// The password may be empty if the MPD server doesn't require authentication.
func (s *MPDServer) Login(username, password string) mediaprovider.LoginResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Test connection
	conn, err := s.dialAndAuth(password)
	if err != nil {
		return mediaprovider.LoginResponse{
			Error:       err,
			IsAuthError: strings.Contains(err.Error(), "authentication"),
		}
	}

	// Initialize the connection pool and semaphore
	s.pool = []*mpd.Client{conn}
	s.password = password
	s.connected = true
	s.connSem = make(chan struct{}, maxPoolSize)
	s.activeConns = 0
	s.provider = &mpdMediaProvider{
		server:            s,
		artistInfoFetcher: newArtistInfoFetcher(s.Language),
	}

	log.Printf("Using MPD connection pool with max %d concurrent connections", maxPoolSize)

	// Pre-warm the connection pool with 2 additional connections
	// This helps with remote connections by establishing them early
	go func() {
		log.Printf("Pre-warming connection pool...")
		for i := 0; i < 2; i++ {
			warmConn, warmErr := s.dialAndAuth(password)
			if warmErr != nil {
				log.Printf("Failed to pre-warm connection %d: %v", i+1, warmErr)
				continue
			}
			s.mu.Lock()
			s.pool = append(s.pool, warmConn)
			s.mu.Unlock()
			log.Printf("Pre-warmed connection %d/%d", i+1, 2)
		}
		log.Printf("Connection pool pre-warming complete")
	}()

	return mediaprovider.LoginResponse{}
}

// dialAndAuth creates a new MPD connection and authenticates if needed.
// For remote connections, this tests connectivity before returning.
func (s *MPDServer) dialAndAuth(password string) (*mpd.Client, error) {
	// Parse hostname to detect remote connections
	host, _, err := net.SplitHostPort(s.Hostname)
	if err != nil {
		// If no port, assume it's just the host
		host = s.Hostname
	}

	isRemote := host != "localhost" && host != "127.0.0.1" && host != "::1"
	if isRemote {
		log.Printf("Connecting to remote MPD server at %s...", s.Hostname)
	}

	conn, err := mpd.Dial("tcp", s.Hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MPD: %w", err)
	}

	if password != "" {
		if err := conn.Command("password %s", password).OK(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("MPD authentication failed: %w", err)
		}
	}

	// For remote connections, test the connection immediately
	if isRemote {
		if err := conn.Ping(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("MPD connection test failed: %w", err)
		}
		log.Printf("Remote MPD connection successful (version: %s)", conn.Version())
	}

	return conn, nil
}

// MediaProvider returns the MediaProvider implementation.
func (s *MPDServer) MediaProvider() mediaprovider.MediaProvider {
	return s.provider
}

// IsJukeboxOnly implements mediaprovider.JukeboxOnlyServer.
// MPD is always jukebox-only — it never provides streaming URLs.
func (s *MPDServer) IsJukeboxOnly() bool {
	return true
}

// JukeboxStart implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxStart() error { return s.provider.JukeboxStart() }

// JukeboxStop implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxStop() error { return s.provider.JukeboxStop() }

// JukeboxSeek implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxSeek(idx, seconds int) error { return s.provider.JukeboxSeek(idx, seconds) }

// JukeboxClear implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxClear() error { return s.provider.JukeboxClear() }

// JukeboxAdd implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxAdd(trackID string) error { return s.provider.JukeboxAdd(trackID) }

// JukeboxRemove implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxRemove(idx int) error { return s.provider.JukeboxRemove(idx) }

// JukeboxGetStatus implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxGetStatus() (*mediaprovider.JukeboxStatus, error) {
	return s.provider.JukeboxGetStatus()
}

// JukeboxSet implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxSet(trackID string) error { return s.provider.JukeboxSet(trackID) }

// JukeboxSetVolume implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxSetVolume(vol int) error { return s.provider.JukeboxSetVolume(vol) }

// JukeboxPlay implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxPlay(idx int) error { return s.provider.JukeboxPlay(idx) }

// JukeboxGetQueue implements mediaprovider.JukeboxProvider by delegating to the provider.
func (s *MPDServer) JukeboxGetQueue() ([]*mediaprovider.Track, int, error) {
	return s.provider.JukeboxGetQueue()
}

// ClearCaches implements mediaprovider.CacheManager by delegating to the provider.
func (s *MPDServer) ClearCaches() {
	if s.provider != nil {
		s.provider.ClearCaches()
	}
}

// reconnect attempts to reconnect to the MPD server.
func (s *MPDServer) reconnect(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconnectLocked(password)
}

// reconnectLocked performs the actual reconnection. Must be called with s.mu held.
func (s *MPDServer) reconnectLocked(password string) error {
	// Close all existing connections
	for _, conn := range s.pool {
		if conn != nil {
			conn.Close()
		}
	}
	s.pool = nil

	// Create a new connection
	conn, err := s.dialAndAuth(password)
	if err != nil {
		s.connected = false
		return err
	}

	s.pool = []*mpd.Client{conn}
	s.connected = true
	return nil
}

// getConn retrieves a connection from the pool or creates a new one.
// Must be called with s.mu held.
func (s *MPDServer) getConnLocked() (*mpd.Client, error) {
	if !s.connected {
		return nil, ErrNotConnected
	}

	// Try to get an existing connection from the pool
	if len(s.pool) > 0 {
		conn := s.pool[len(s.pool)-1]
		s.pool = s.pool[:len(s.pool)-1]
		s.activeConns++
		return conn, nil
	}

	// Pool is empty - check if we can create a new connection
	if s.activeConns >= maxPoolSize {
		// All connections are in use - this shouldn't happen with semaphore
		// but handle it gracefully
		return nil, fmt.Errorf("connection pool exhausted (%d active)", s.activeConns)
	}

	// Create a new connection
	conn, err := s.dialAndAuth(s.password)
	if err != nil {
		return nil, fmt.Errorf("failed to create new connection: %w", err)
	}

	s.activeConns++
	log.Printf("Created new MPD connection (active: %d/%d)", s.activeConns, maxPoolSize)
	return conn, nil
}

// returnConn returns a connection to the pool.
// Must be called with s.mu held.
func (s *MPDServer) returnConnLocked(conn *mpd.Client) {
	if conn == nil {
		return
	}

	s.activeConns--

	// Don't return to pool if we're over the limit
	if len(s.pool) >= maxPoolSize {
		conn.Close()
		log.Printf("Closed excess connection (active: %d/%d)", s.activeConns, maxPoolSize)
		return
	}

	s.pool = append(s.pool, conn)
}

// withConn executes a function with an MPD connection from the pool.
// Connections are acquired from a pool to allow concurrent operations.
func (s *MPDServer) withConn(fn func(*mpd.Client) error) error {
	// Acquire semaphore slot (blocks if all connections are in use)
	s.connSem <- struct{}{}
	defer func() { <-s.connSem }()

	// Get a connection from the pool
	s.mu.Lock()
	conn, err := s.getConnLocked()
	s.mu.Unlock()

	if err != nil {
		return err
	}

	// Ping the connection first to ensure it's alive (important for remote/internet connections)
	if pingErr := conn.Ping(); pingErr != nil {
		log.Printf("MPD connection ping failed: %v, reconnecting...", pingErr)
		conn.Close()
		s.mu.Lock()
		s.activeConns--
		s.mu.Unlock()

		// Reconnect
		if reconnErr := s.reconnect(s.password); reconnErr != nil {
			return reconnErr
		}

		// Get fresh connection
		s.mu.Lock()
		conn, err = s.getConnLocked()
		s.mu.Unlock()

		if err != nil {
			return err
		}
	}

	// Execute the function (without holding the mutex to allow concurrency)
	err = fn(conn)

	// Handle connection errors
	if err != nil && isConnectionError(err) {
		log.Printf("MPD connection error: %v", err)
		// Close the bad connection
		conn.Close()
		s.mu.Lock()
		s.activeConns-- // Decrement since we're not returning it
		s.mu.Unlock()

		// Try to reconnect and retry once
		if reconnErr := s.reconnect(s.password); reconnErr != nil {
			return reconnErr
		}

		// Get a new connection and retry
		s.mu.Lock()
		conn, err = s.getConnLocked()
		s.mu.Unlock()

		if err != nil {
			return err
		}

		err = fn(conn)
	}

	// Return the connection to the pool (or close it if there was an error)
	s.mu.Lock()
	if err != nil && isConnectionError(err) {
		conn.Close()
		s.activeConns-- // Decrement since we're not returning it
	} else {
		s.returnConnLocked(conn)
	}
	s.mu.Unlock()

	return err
}

// isConnectionError checks if an error indicates a lost network connection.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for standard network/io sentinel errors
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	// Check for net.Error (timeout, temporary failures)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// Fallback: check for "broken pipe" and "connection reset" which aren't
	// wrapped in net.Error on all platforms
	errStr := err.Error()
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "use of closed network connection")
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
