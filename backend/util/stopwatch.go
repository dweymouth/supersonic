package util

import "time"

type Stopwatch struct {
	running bool
	started time.Time
	elapsed time.Duration
}

func (s *Stopwatch) Start() {
	if s.running {
		return
	}
	s.started = time.Now()
	s.running = true
}

func (s *Stopwatch) Stop() {
	if !s.running {
		return
	}
	s.elapsed += time.Since(s.started)
	s.running = false
}

func (s *Stopwatch) Elapsed() time.Duration {
	e := s.elapsed
	if s.running {
		e += time.Since(s.started)
	}
	return e
}

func (s *Stopwatch) Reset() {
	s.running = false
	s.elapsed = time.Duration(0)
}
