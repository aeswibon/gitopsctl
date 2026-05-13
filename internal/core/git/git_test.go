package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func createDummyRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "dummy-repo")
	if err != nil {
		t.Fatal(err)
	}

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "test")
	runGit("config", "commit.gpgsign", "false")

	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatal(err)
	}

	runGit("add", ".")
	runGit("commit", "-m", "initial commit")
	runGit("branch", "-M", "master")

	return dir
}

func TestCloneOrPull(t *testing.T) {
	logger := zap.NewNop()
	repoDir := createDummyRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	targetDir, err := os.MkdirTemp("", "target-repo")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(targetDir) }()

	// 1. Test Clone
	hash, err := CloneOrPull(context.Background(), logger, repoDir, "master", targetDir, AuthOptions{})
	if err != nil {
		t.Fatalf("CloneOrPull() error = %v", err)
	}
	if hash == "" {
		t.Error("Expected commit hash, got empty string")
	}

	// 2. Test Pull (Already up to date)
	hash2, err := CloneOrPull(context.Background(), logger, repoDir, "master", targetDir, AuthOptions{})
	if err != nil {
		t.Fatalf("CloneOrPull() (pull) error = %v", err)
	}
	if hash != hash2 {
		t.Errorf("Expected same hash %s, got %s", hash, hash2)
	}
}

func TestGetLatestCommitHash(t *testing.T) {
	logger := zap.NewNop()
	repoDir := createDummyRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	hash, err := GetLatestCommitHash(logger, repoDir)
	if err != nil {
		t.Fatalf("GetLatestCommitHash() error = %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("Expected 40 char hash, got %d chars: %s", len(hash), hash)
	}
}

func TestTempRepoDir(t *testing.T) {
	dir, err := CreateTempRepoDir()
	if err != nil {
		t.Fatalf("CreateTempRepoDir() error = %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Expected temp dir to exist")
	}
}

func TestSetupAuth_Scenarios(t *testing.T) {
	// 1. Token auth
	auth := setupAuth("https://github.com/org/repo.git", AuthOptions{Token: "test-token"})
	if auth == nil {
		t.Fatal("expected non-nil auth for token")
	}
	if !strings.Contains(auth.String(), "oauth2") {
		t.Errorf("unexpected auth string: %s", auth.String())
	}

	// 2. Basic auth
	auth = setupAuth("https://github.com/org/repo.git", AuthOptions{Username: "user", Password: "pass"})
	if auth == nil {
		t.Fatal("expected non-nil auth for basic auth")
	}
	if !strings.Contains(auth.String(), "user") {
		t.Errorf("unexpected auth string: %s", auth.String())
	}
}

func TestCloneOrPull_Errors(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	tmpDir, _ := os.MkdirTemp("", "git-error")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// 1. Invalid URL
	_, err := CloneOrPull(ctx, logger, "not-a-url", "master", tmpDir, AuthOptions{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// 2. Invalid branch
	repoDir := createDummyRepo(t)
	defer func() { _ = os.RemoveAll(repoDir) }()

	targetDir, _ := os.MkdirTemp("", "git-branch-error")
	defer func() { _ = os.RemoveAll(targetDir) }()

	_, err = CloneOrPull(ctx, logger, repoDir, "invalid-branch", targetDir, AuthOptions{})
	if err == nil {
		t.Error("expected error for invalid branch")
	}
}

func TestCleanUpRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cleanup-repo")
	if err != nil {
		t.Fatal(err)
	}
	if err := CleanUpRepo(zap.NewNop(), tmpDir); err != nil {
		t.Fatalf("CleanUpRepo() error = %v", err)
	}
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Fatalf("expected cleaned directory to be removed")
	}
}
