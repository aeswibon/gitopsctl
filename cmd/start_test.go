package cmd

import "testing"

func TestStartCommand_Configured(t *testing.T) {
	if startCmd == nil || startCmd.Use == "" {
		t.Fatal("startCmd should be configured")
	}
}
