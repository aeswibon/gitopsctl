package utils

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddListFlags(t *testing.T) {
	cmd := &cobra.Command{}
	opts := &ListOptions{}
	AddListFlags(cmd, opts, "name")

	if cmd.Flag("output") == nil {
		t.Error("Expected 'output' flag to be added")
	}
	if cmd.Flag("details") == nil {
		t.Error("Expected 'details' flag to be added")
	}

	// Verify default sort
	if opts.SortBy != "name" {
		t.Errorf("Expected default sort 'name', got %s", opts.SortBy)
	}
}
