package cmd

import "testing"

func TestCommandGroups_Configured(t *testing.T) {
	if appGroup == nil || appGroup.ID == "" {
		t.Fatal("appGroup should be configured")
	}
	if clusterGroup == nil || clusterGroup.ID == "" {
		t.Fatal("clusterGroup should be configured")
	}
}
