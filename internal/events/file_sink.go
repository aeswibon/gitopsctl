package events

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// FileSink appends one JSON object per line (JSONL).
type FileSink struct {
	mu   sync.Mutex
	path string
}

// NewFileSink creates a sink that appends to path (creates parent dirs as needed).
func NewFileSink(path string) (*FileSink, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return &FileSink{path: path}, nil
}

// Write appends a single NDJSON record.
func (s *FileSink) Write(ctx context.Context, e *Envelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	line, err := MarshalJSONLine(e)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(line); err != nil {
		return err
	}
	return f.Sync()
}
