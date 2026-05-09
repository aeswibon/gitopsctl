package events

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestBus_Emit(t *testing.T) {
	logger := zap.NewNop()
	sink := NewStreamSink()
	bus := NewBus(logger, "test-source", sink)

	eventsCh, unsubscribe := sink.Subscribe(10)
	defer unsubscribe()

	bus.Emit(context.Background(), TypeAppSyncSucceeded, map[string]any{"app": "test"})

	select {
	case env := <-eventsCh:
		if string(env.Type) != string(TypeAppSyncSucceeded) {
			t.Errorf("Expected TypeAppSyncSucceeded, got %v", env.Type)
		}

		if env.Data["app"] != "test" {
			t.Errorf("Expected app=test, got %v", env.Data["app"])
		}
	case <-time.After(time.Second):
		t.Error("No event received within timeout")
	}
}
