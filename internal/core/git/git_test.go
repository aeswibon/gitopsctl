package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	hash, err := CloneOrPull(context.Background(), logger, repoDir, "master", targetDir)
	if err != nil {
		t.Fatalf("CloneOrPull() error = %v", err)
	}
	if hash == "" {
		t.Error("Expected commit hash, got empty string")
	}

	// 2. Test Pull (Already up to date)
	hash2, err := CloneOrPull(context.Background(), logger, repoDir, "master", targetDir)
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
