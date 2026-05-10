package events

import (
	"context"
	"sync"
)

// HistorySink keeps a fixed-size ring buffer of the most recent events in memory.
type HistorySink struct {
	mu     sync.RWMutex
	size   int
	events []*Envelope
	cursor int
	count  int
}

// NewHistorySink creates a sink that retains up to size events.
func NewHistorySink(size int) *HistorySink {
	if size <= 0 {
		size = 100
	}
	return &HistorySink{
		size:   size,
		events: make([]*Envelope, size),
	}
}

// Write adds an event to the history buffer.
func (s *HistorySink) Write(ctx context.Context, e *Envelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[s.cursor] = e
	s.cursor = (s.cursor + 1) % s.size
	if s.count < s.size {
		s.count++
	}
	return nil
}

// All returns all currently stored events, ordered from oldest to newest.
func (s *HistorySink) All() []*Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Envelope, 0, s.count)
	if s.count < s.size {
		// Buffer is not full yet, start from index 0
		for i := 0; i < s.count; i++ {
			result = append(result, s.events[i])
		}
	} else {
		// Buffer is full, start from cursor (which points to the oldest)
		for i := 0; i < s.size; i++ {
			idx := (s.cursor + i) % s.size
			result = append(result, s.events[idx])
		}
	}
	return result
}
