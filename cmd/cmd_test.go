package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func TestRootCommand(t *testing.T) {
	root := RootCmd()
	output, err := executeCommand(root, "--help")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output == "" {
		t.Error("Expected help output, got empty string")
	}
}

func TestListAppsCommand(t *testing.T) {
	root := RootCmd()
	_, err := executeCommand(root, "list-apps")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListClustersCommand(t *testing.T) {
	root := RootCmd()
	_, err := executeCommand(root, "list-clusters")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestAppStatusCommand(t *testing.T) {
	root := RootCmd()
	_, err := executeCommand(root, "app-status", "non-existent")
	// This might return an error because the app doesn't exist, which is fine
	if err == nil {
		t.Error("Expected error for non-existent app status")
	}
}

func TestRegisterAppCommand_Help(t *testing.T) {
	root := RootCmd()
	_, err := executeCommand(root, "register-apps", "--help")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
