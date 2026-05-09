package cmd

import "testing"

func TestApproveAppCommand_Configured(t *testing.T) {
	if approveAppCmd == nil || approveAppCmd.Use == "" {
		t.Fatal("approveAppCmd should be configured")
	}
}
