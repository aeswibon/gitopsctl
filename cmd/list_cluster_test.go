package cmd

import "testing"

func TestListClusterCommand_Configured(t *testing.T) {
	if listClusterCmd == nil || listClusterCmd.Use == "" {
		t.Fatal("listClusterCmd should be configured")
	}
}
