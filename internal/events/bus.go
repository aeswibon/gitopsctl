package events

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Emitter emits typed integration events.
type Emitter interface {
	Emit(ctx context.Context, typ Type, data map[string]any)
}

// Bus fans out to sinks asynchronously (does not block reconcilers for slow webhooks).
type Bus struct {
	logger *zap.Logger
	source string
	sinks  []Sink
}

// NewBus builds an emitter with one or more sinks.
func NewBus(logger *zap.Logger, source string, sinks ...Sink) *Bus {
	return &Bus{logger: logger, source: source, sinks: sinks}
}

func cloneEnvelope(e *Envelope) *Envelope {
	dataCopy := make(map[string]any, len(e.Data))
	for k, v := range e.Data {
		dataCopy[k] = v
	}
	cp := *e
	cp.Data = dataCopy
	return &cp
}

// Emit delivers the event to all sinks in separate goroutines (best-effort).
func (b *Bus) Emit(ctx context.Context, typ Type, data map[string]any) {
	if len(b.sinks) == 0 {
		return
	}
	env := NewEnvelope(b.source, typ, data)
	for _, s := range b.sinks {
		sink := s
		payload := cloneEnvelope(env)
		go func() {
			cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := sink.Write(cctx, payload); err != nil && b.logger != nil {
				b.logger.Warn("event sink write failed",
					zap.String("type", string(typ)),
					zap.Error(err))
			}
		}()
	}
}

// Noop is an Emitter that drops all events.
type Noop struct{}

func (Noop) Emit(context.Context, Type, map[string]any) {}
