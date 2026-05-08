package cmd

import "testing"

func TestUnregisterClusterCommand_Configured(t *testing.T) {
	if unregisterClusterCmd == nil || unregisterClusterCmd.Use == "" {
		t.Fatal("unregisterClusterCmd should be configured")
	}
}
