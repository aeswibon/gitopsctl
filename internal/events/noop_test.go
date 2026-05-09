package events

import (
	"context"
	"testing"
)

func TestNoopEmit_DoesNothing(t *testing.T) {
	var n Noop
	n.Emit(context.Background(), TypeAppRegistered, map[string]any{"app": "a1"})
}

func TestNewWebhookSink_Basic(t *testing.T) {
	s := NewWebhookSink("https://example.com/hook", "token")
	if s == nil {
		t.Fatal("expected webhook sink instance")
	}
}
