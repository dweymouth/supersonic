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

const (
	maxPoolSize     = 5 // Maximum number of concurrent connections (MPD default limit is often 10)
	pingIdleTimeout = 30 * time.Second
)

type pooledConn struct {
	client   *mpd.Client
	lastUsed time.Time
}

// MPDServer implements mediaprovider.Server for MPD connections.
type MPDServer struct {
	Hostname string

	mu          sync.Mutex // Protects the pool and connection state
	pool        []*pooledConn
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
	s.pool = []*pooledConn{{client: conn, lastUsed: time.Now()}}
	s.password = password
	s.connected = true
	s.connSem = make(chan struct{}, maxPoolSize)
	s.activeConns = 0
	s.provider = &mpdMediaProvider{
		server: s,
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
			s.pool = append(s.pool, &pooledConn{client: warmConn, lastUsed: time.Now()})
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

// reconnect attempts to reconnect to the MPD server.
func (s *MPDServer) reconnect(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconnectLocked(password)
}

// reconnectLocked performs the actual reconnection. Must be called with s.mu held.
func (s *MPDServer) reconnectLocked(password string) error {
	// Close all existing connections
	for _, pc := range s.pool {
		if pc != nil {
			pc.client.Close()
		}
	}
	s.pool = nil

	conn, err := s.dialAndAuth(password)
	if err != nil {
		s.connected = false
		return err
	}

	s.pool = []*pooledConn{{client: conn, lastUsed: time.Now()}}
	s.connected = true
	return nil
}

func (s *MPDServer) getConnLocked() (*pooledConn, error) {
	if !s.connected {
		return nil, ErrNotConnected
	}

	if len(s.pool) > 0 {
		pc := s.pool[len(s.pool)-1]
		s.pool = s.pool[:len(s.pool)-1]
		s.activeConns++
		return pc, nil
	}

	// Pool is empty - check if we can create a new connection
	if s.activeConns >= maxPoolSize {
		// All connections are in use - this shouldn't happen with semaphore
		// but handle it gracefully
		return nil, fmt.Errorf("connection pool exhausted (%d active)", s.activeConns)
	}

	conn, err := s.dialAndAuth(s.password)
	if err != nil {
		return nil, fmt.Errorf("failed to create new connection: %w", err)
	}

	s.activeConns++
	log.Printf("Created new MPD connection (active: %d/%d)", s.activeConns, maxPoolSize)
	return &pooledConn{client: conn, lastUsed: time.Now()}, nil
}

func (s *MPDServer) returnConnLocked(pc *pooledConn) {
	if pc == nil {
		return
	}

	s.activeConns--

	if len(s.pool) >= maxPoolSize {
		pc.client.Close()
		log.Printf("Closed excess connection (active: %d/%d)", s.activeConns, maxPoolSize)
		return
	}

	pc.lastUsed = time.Now()
	s.pool = append(s.pool, pc)
}

func (s *MPDServer) withConn(fn func(*mpd.Client) error) error {
	s.connSem <- struct{}{}
	defer func() { <-s.connSem }()

	s.mu.Lock()
	pc, err := s.getConnLocked()
	s.mu.Unlock()

	if err != nil {
		return err
	}

	if time.Since(pc.lastUsed) > pingIdleTimeout {
		if pingErr := pc.client.Ping(); pingErr != nil {
			log.Printf("MPD connection ping failed: %v, reconnecting...", pingErr)
			pc.client.Close()
			s.mu.Lock()
			s.activeConns--
			s.mu.Unlock()

			if reconnErr := s.reconnect(s.password); reconnErr != nil {
				return reconnErr
			}

			s.mu.Lock()
			pc, err = s.getConnLocked()
			s.mu.Unlock()

			if err != nil {
				return err
			}
		}
	}

	err = fn(pc.client)

	if err != nil && isConnectionError(err) {
		log.Printf("MPD connection error: %v", err)
		pc.client.Close()
		s.mu.Lock()
		s.activeConns--
		s.mu.Unlock()

		if reconnErr := s.reconnect(s.password); reconnErr != nil {
			return reconnErr
		}

		s.mu.Lock()
		pc, err = s.getConnLocked()
		s.mu.Unlock()

		if err != nil {
			return err
		}

		err = fn(pc.client)
	}

	s.mu.Lock()
	if err != nil && isConnectionError(err) {
		pc.client.Close()
		s.activeConns--
	} else {
		s.returnConnLocked(pc)
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
