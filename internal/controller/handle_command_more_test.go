package controller

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"go.uber.org/zap"
)

func TestHandleAppCommand_StartMissingClusterUpdatesApp(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)

	tmpDir, _ := os.MkdirTemp("", "handle-start")
	defer os.RemoveAll(tmpDir)
	appCfg := filepath.Join(tmpDir, "apps.json")

	a := &app.Application{
		Name:            "orphan",
		RepoURL:         "https://github.com/example/repo.git",
		ClusterName:     "missing",
		PollingInterval: time.Minute,
		Status:          "Pending",
	}
	apps.Add(a)

	ctrl.handleAppCommand(AppCommand{Type: AppCommandStart, AppName: "orphan"}, appCfg)

	apps.RLock()
	got, _ := apps.Get("orphan")
	apps.RUnlock()
	if got == nil || got.Status != "Error" {
		t.Fatalf("expected Error status for missing cluster, got %+v", got)
	}
}

func TestHandleAppCommand_StopNonRunning_LogsOnly(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)
	appCfg := filepath.Join(t.TempDir(), "apps.json")

	ctrl.handleAppCommand(AppCommand{Type: AppCommandStop, AppName: "none"}, appCfg)
}

func TestHandleAppCommand_SyncNonRunning_LogsOnly(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)
	appCfg := filepath.Join(t.TempDir(), "apps.json")

	ctrl.handleAppCommand(AppCommand{Type: AppCommandSync, AppName: "none"}, appCfg)
}

func TestHandleAppCommand_ApproveInvalidCommit_NoPanic(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	apps.Add(&app.Application{Name: "a1", Interval: "1m"})
	ctrl := NewController(logger, apps, cluster.NewClusters())
	appCfg := filepath.Join(t.TempDir(), "apps.json")

	ctrl.handleAppCommand(AppCommand{
		Type:    AppCommandApprove,
		AppName: "a1",
		Data:    map[string]any{"commitHash": ""},
	}, appCfg)

	ctrl.handleAppCommand(AppCommand{
		Type:    AppCommandApprove,
		AppName: "missing",
		Data:    map[string]any{"commitHash": "abc"},
	}, appCfg)
}
