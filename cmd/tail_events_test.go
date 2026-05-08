package cmd

import "testing"

func TestTailEventsCommand_Configured(t *testing.T) {
	if tailEventsCmd == nil || tailEventsCmd.Use == "" {
		t.Fatal("tailEventsCmd should be configured")
	}
}
