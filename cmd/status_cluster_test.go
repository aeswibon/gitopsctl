package cmd

import "testing"

func TestStatusClusterCommand_Configured(t *testing.T) {
	if statusClusterCmd == nil || statusClusterCmd.Use == "" {
		t.Fatal("statusClusterCmd should be configured")
	}
}
