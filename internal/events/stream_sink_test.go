package events

import (
	"context"
	"testing"
	"time"
)

func TestStreamSink_Lifecycle(t *testing.T) {
	sink := NewStreamSink()

	ch, unsubscribe := sink.Subscribe(10)

	env := NewEnvelope("test", TypeAppSyncSucceeded, nil)

	if err := sink.Write(context.Background(), env); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	select {
	case <-ch:
		// success
	case <-time.After(time.Second):
		t.Error("Did not receive event")
	}

	unsubscribe()

	// Should not panic or error if writing after unsubscribe (though nobody is listening)
	_ = sink.Write(context.Background(), env)
}
