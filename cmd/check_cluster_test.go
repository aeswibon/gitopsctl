package cmd

import "testing"

func TestCheckClusterCommand_Configured(t *testing.T) {
	if checkClusterCmd == nil || checkClusterCmd.Use == "" {
		t.Fatal("checkClusterCmd should be configured")
	}
}
