package util

import (
	"sync"
	"time"
)

// Stopwatch is a thread-safe timer for measuring elapsed time.
// It can be started, stopped, and reset, and supports reading
// the elapsed time while running or stopped.
type Stopwatch struct {
	mu      sync.Mutex
	running bool
	started time.Time
	elapsed time.Duration
}

// Start begins or resumes the stopwatch.
// If already running, this is a no-op.
func (s *Stopwatch) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	s.started = time.Now()
	s.running = true
}

// Stop pauses the stopwatch and accumulates the elapsed time.
// If already stopped, this is a no-op.
func (s *Stopwatch) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.elapsed += time.Since(s.started)
	s.running = false
}

// Elapsed returns the total elapsed time.
// If the stopwatch is running, includes time since last Start().
// Safe to call concurrently with other methods.
func (s *Stopwatch) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := s.elapsed
	if s.running {
		e += time.Since(s.started)
	}
	return e
}

// Reset stops the stopwatch and clears the elapsed time.
func (s *Stopwatch) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	s.elapsed = time.Duration(0)
}
