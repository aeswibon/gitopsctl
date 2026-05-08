package cmd

import "testing"

func TestCommandEmitter_AlwaysReturnsEmitter(t *testing.T) {
	if commandEmitter() == nil {
		t.Fatal("expected commandEmitter to return a non-nil emitter")
	}
}
