package events

import (
	"context"
	"sync"
)

// StreamSink keeps in-memory subscribers for live event streaming (SSE/WebSocket adapters).
type StreamSink struct {
	mu       sync.RWMutex
	nextID   int
	channels map[int]chan *Envelope
}

// NewStreamSink creates a stream sink for in-process subscribers.
func NewStreamSink() *StreamSink {
	return &StreamSink{
		channels: make(map[int]chan *Envelope),
	}
}

func cloneForStream(e *Envelope) *Envelope {
	dataCopy := make(map[string]any, len(e.Data))
	for k, v := range e.Data {
		dataCopy[k] = v
	}
	cp := *e
	cp.Data = dataCopy
	return &cp
}

// Write fan-outs events to subscribers. Slow subscribers drop events (best-effort).
func (s *StreamSink) Write(ctx context.Context, e *Envelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload := cloneForStream(e)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.channels {
		select {
		case ch <- payload:
		default:
			// Drop for slow consumers; stream clients must handle gaps.
		}
	}
	return nil
}

// Subscribe returns a channel and an unsubscribe function.
func (s *StreamSink) Subscribe(buffer int) (<-chan *Envelope, func()) {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan *Envelope, buffer)

	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.channels[id] = ch
	s.mu.Unlock()

	unsubscribe := func() {
		s.mu.Lock()
		if existing, ok := s.channels[id]; ok {
			delete(s.channels, id)
			close(existing)
		}
		s.mu.Unlock()
	}

	return ch, unsubscribe
}
