package cmd

import "testing"

func TestListClusterCommand_Configured(t *testing.T) {
	if listClusterCmd == nil || listClusterCmd.Use == "" {
		t.Fatal("listClusterCmd should be configured")
	}
}

func TestHandleEmptyClustersForList(t *testing.T) {
	if err := handleEmptyClustersForList(""); err != nil {
		t.Fatal(err)
	}
	if err := handleEmptyClustersForList("active"); err != nil {
		t.Fatal(err)
	}
}
