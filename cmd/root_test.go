package cmd

import "testing"

func TestRootCommand_Configured(t *testing.T) {
	if rootCmd == nil || rootCmd.Use == "" {
		t.Fatal("rootCmd should be configured")
	}
}
