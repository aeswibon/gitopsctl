package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEnvelope(t *testing.T) {
	e := NewEnvelope("unit-test", TypeControllerStarted, map[string]any{"applications": 1})
	if e.SpecVersion != SpecVersion {
		t.Fatalf("specversion: got %q", e.SpecVersion)
	}
	if e.Source != "unit-test" {
		t.Fatalf("source: got %q", e.Source)
	}
	if e.Type != string(TypeControllerStarted) {
		t.Fatalf("type: got %q", e.Type)
	}
	if e.ID == "" {
		t.Fatal("expected id")
	}
	if e.Time.IsZero() {
		t.Fatal("expected time")
	}
	if e.Data["applications"] != 1 {
		t.Fatalf("data: %#v", e.Data)
	}
}

func TestMarshalJSONLine(t *testing.T) {
	e := NewEnvelope("test", TypeAppSyncSucceeded, map[string]any{"app": "a", "commit": "abc"})
	e.Time = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	line, err := MarshalJSONLine(e)
	if err != nil {
		t.Fatal(err)
	}
	if line[len(line)-1] != '\n' {
		t.Fatalf("expected trailing newline, got %q", line)
	}
	var back Envelope
	if err := json.Unmarshal(line[:len(line)-1], &back); err != nil {
		t.Fatal(err)
	}
	if back.Data["app"] != "a" {
		t.Fatalf("round-trip data: %#v", back.Data)
	}
}
