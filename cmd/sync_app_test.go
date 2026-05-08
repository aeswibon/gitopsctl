package cmd

import "testing"

func TestSyncAppCommand_Configured(t *testing.T) {
	if syncAppCmd == nil || syncAppCmd.Use == "" {
		t.Fatal("syncAppCmd should be configured")
	}
}
