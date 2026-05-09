package cmd

import "testing"

func TestStatusAppCommand_Configured(t *testing.T) {
	if statusAppCmd == nil || statusAppCmd.Use == "" {
		t.Fatal("statusAppCmd should be configured")
	}
}
