package events

import (
	"context"
)

// Sink receives envelopes (file append, HTTP webhook, etc.).
type Sink interface {
	Write(ctx context.Context, e *Envelope) error
}
