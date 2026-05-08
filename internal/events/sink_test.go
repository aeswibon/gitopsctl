package events

import (
	"context"
	"testing"
)

type testSink struct{}

func (testSink) Write(_ context.Context, _ *Envelope) error { return nil }

func TestSinkInterface_IsImplemented(t *testing.T) {
	var s Sink = testSink{}
	if err := s.Write(context.Background(), &Envelope{}); err != nil {
		t.Fatalf("expected no error writing envelope, got %v", err)
	}
}
