package events

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSink_Write(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-sink-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	logFile := filepath.Join(tmpDir, "events.log")
	sink, _ := NewFileSink(logFile)

	env := NewEnvelope("test", TypeAppSyncSucceeded, map[string]any{"foo": "bar"})

	if err := sink.Write(context.Background(), env); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file is empty")
	}
}
