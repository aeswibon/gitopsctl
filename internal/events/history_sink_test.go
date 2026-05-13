package events

import (
	"context"
	"testing"
)

func TestHistorySink(t *testing.T) {
	sink := NewHistorySink(2)
	ctx := context.Background()

	e1 := &Envelope{Type: "t1"}
	e2 := &Envelope{Type: "t2"}
	e3 := &Envelope{Type: "t3"}

	_ = sink.Write(ctx, e1)
	_ = sink.Write(ctx, e2)

	history := sink.All()
	if len(history) != 2 {
		t.Fatalf("expected 2 events, got %d", len(history))
	}
	if history[0].Type != "t1" || history[1].Type != "t2" {
		t.Error("unexpected history order")
	}

	_ = sink.Write(ctx, e3) // Should drop e1
	history = sink.All()
	if len(history) != 2 {
		t.Fatal("expected 2 events after overflow")
	}
	if history[0].Type != "t2" || history[1].Type != "t3" {
		t.Error("unexpected history after overflow")
	}
}
