package events

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type countingSink struct {
	n atomic.Int32
}

func (c *countingSink) Write(ctx context.Context, e *Envelope) error {
	c.n.Add(1)
	return nil
}

func TestBus_Emit_FansOutToSinks(t *testing.T) {
	var s countingSink
	bus := NewBus(zap.NewNop(), "unit-test", &s)
	bus.Emit(context.Background(), TypeAppRegistered, map[string]any{"app": "x"})

	deadline := time.Now().Add(2 * time.Second)
	for s.n.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.n.Load() != 1 {
		t.Fatalf("expected sink write, got %d", s.n.Load())
	}
}

func TestNoopEmit_InterfaceSatifiesEmitter(t *testing.T) {
	var e Emitter = Noop{}
	e.Emit(context.Background(), TypeAppRegistered, map[string]any{"k": "v"})
}
