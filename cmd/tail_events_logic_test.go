package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunTailEvents_FromStartNoFollow(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "events-tail")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := filepath.Join(tmpDir, "events.jsonl")
	if err := os.WriteFile(p, []byte("{\"id\":1}\n"), 0644); err != nil {
		t.Fatalf("failed writing events file: %v", err)
	}

	tailEventsFile = p
	tailEventsFromStart = true
	tailEventsFollow = false
	tailEventsPoll = 5 * time.Millisecond

	if err := runTailEvents(nil, nil); err != nil {
		t.Fatalf("runTailEvents() error = %v", err)
	}
}

func TestRunTailEvents_OpenError(t *testing.T) {
	tailEventsFile = filepath.Join(t.TempDir(), "does-not-exist.jsonl")
	tailEventsFollow = false
	tailEventsFromStart = true

	if err := runTailEvents(nil, nil); err == nil {
		t.Fatal("expected error when events file is missing")
	}
}
