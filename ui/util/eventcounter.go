package util

import (
	"time"
)

type EventCounter struct {
	buf []time.Time
	ptr int
}

func NewEventCounter(maxN int) *EventCounter {
	buffer := make([]time.Time, maxN)
	return &EventCounter{
		buf: buffer,
	}
}

func (e *EventCounter) Add() {
	e.buf[e.ptr] = time.Now()
	e.ptr = (e.ptr + 1) % len(e.buf)
}

func (e *EventCounter) NumEventsSince(t time.Time) int {
	i := e.ptr - 1

	count := 0
	for {
		if i < 0 {
			i = len(e.buf) - 1
		}
		if !e.buf[i].IsZero() && e.buf[i].After(t) {
			count++
		} else {
			break
		}
		if count == len(e.buf) {
			break
		}
		i--
	}
	return count
}
