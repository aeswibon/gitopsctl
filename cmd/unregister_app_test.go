package cmd

import "testing"

func TestUnregisterAppCommand_Configured(t *testing.T) {
	if unregisterAppCmd == nil || unregisterAppCmd.Use == "" {
		t.Fatal("unregisterAppCmd should be configured")
	}
}
