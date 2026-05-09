package cmd

import "testing"

func TestRegisterClusterCommand_Configured(t *testing.T) {
	if registerClusterCmd == nil || registerClusterCmd.Use == "" {
		t.Fatal("registerClusterCmd should be configured")
	}
}
