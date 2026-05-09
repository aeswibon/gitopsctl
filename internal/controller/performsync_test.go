package controller

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	gitcore "aeswibon.com/github/gitopsctl/internal/core/git"
	"go.uber.org/zap"
)

func TestPerformSync_GitFailureSetsError(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)

	tmpDir, _ := os.MkdirTemp("", "perform-sync")
	defer os.RemoveAll(tmpDir)
	appCfg := filepath.Join(tmpDir, "apps.json")

	a := &app.Application{
		Name:            "a1",
		RepoURL:         "not-a-valid-git-url",
		Branch:          "main",
		Path:            "manifests",
		ClusterName:     "c1",
		PollingInterval: time.Minute,
		Status:          "Pending",
	}
	apps.Add(a)

	ctrl.performSync(context.Background(), logger, a, tmpDir, nil, appCfg, "manual")

	if a.Status != "Error" {
		t.Fatalf("expected Error status after git failure, got %s", a.Status)
	}
	if a.ConsecutiveFailures == 0 {
		t.Fatal("expected consecutive failures to increment")
	}
}

func createLocalRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "controller-repo")
	if err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "cm.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	run("branch", "-M", "master")
	return dir
}

func TestPerformSync_ManualPolicySetsOutOfSync(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	ctrl := NewController(logger, apps, cluster.NewClusters())

	repoPath := createLocalRepo(t)
	defer os.RemoveAll(repoPath)
	workDir, _ := os.MkdirTemp("", "perform-sync-manual")
	defer os.RemoveAll(workDir)
	appCfg := filepath.Join(workDir, "apps.json")

	a := &app.Application{
		Name:            "a1",
		RepoURL:         repoPath,
		Branch:          "master",
		Path:            ".",
		ClusterName:     "c1",
		PollingInterval: time.Minute,
		SyncPolicy:      "manual",
		ApprovedGitHash: "different",
		Status:          "Pending",
	}
	apps.Add(a)

	ctrl.performSync(context.Background(), logger, a, workDir, nil, appCfg, "manual")
	if a.Status != "OutOfSync" {
		t.Fatalf("expected OutOfSync, got %s", a.Status)
	}
}

func TestPerformSync_NoChangesSetsSynced(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	ctrl := NewController(logger, apps, cluster.NewClusters())

	repoPath := createLocalRepo(t)
	defer os.RemoveAll(repoPath)
	workDir, _ := os.MkdirTemp("", "perform-sync-nochanges")
	defer os.RemoveAll(workDir)
	appCfg := filepath.Join(workDir, "apps.json")

	// First clone to workDir and read hash.
	hash, err := gitcore.CloneOrPull(context.Background(), logger, repoPath, "master", workDir)
	if err != nil {
		t.Fatalf("initial clone failed: %v", err)
	}

	a := &app.Application{
		Name:              "a1",
		RepoURL:           repoPath,
		Branch:            "master",
		Path:              ".",
		ClusterName:       "c1",
		PollingInterval:   time.Minute,
		LastSyncedGitHash: hash,
		Status:            "Pending",
	}
	apps.Add(a)

	ctrl.performSync(context.Background(), logger, a, workDir, nil, appCfg, "manual")
	if a.Status != "Synced" {
		t.Fatalf("expected Synced on no-change manual sync, got %s", a.Status)
	}
}

func TestPerformSync_ManifestPathMissingSetsError(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	ctrl := NewController(logger, apps, cluster.NewClusters())

	repoPath := createLocalRepo(t)
	defer os.RemoveAll(repoPath)
	workDir, _ := os.MkdirTemp("", "perform-sync-missing-manifest")
	defer os.RemoveAll(workDir)
	appCfg := filepath.Join(workDir, "apps.json")

	if _, err := gitcore.CloneOrPull(context.Background(), logger, repoPath, "master", workDir); err != nil {
		t.Fatalf("clone failed: %v", err)
	}

	headHash, err := gitcore.GetLatestCommitHash(logger, repoPath)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	a := &app.Application{
		Name:              "a1",
		RepoURL:           repoPath,
		Branch:            "master",
		Path:              "does-not-exist",
		ClusterName:       "c1",
		PollingInterval:   time.Minute,
		LastSyncedGitHash: "",
		SyncPolicy:        "auto",
		Status:            "Pending",
	}
	apps.Add(a)

	ctrl.performSync(context.Background(), logger, a, workDir, nil, appCfg, "manual")

	if headHash == "" {
		t.Fatal("expected non-empty repo head hash")
	}
	if a.Status != "Error" {
		t.Fatalf("expected Error for missing manifest path, got %s (msg=%q)", a.Status, a.Message)
	}
	if !strings.Contains(a.Message, "Manifests path") && !strings.Contains(a.Message, "not found") {
		t.Fatalf("unexpected message: %q", a.Message)
	}
}
