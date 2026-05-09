package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"go.uber.org/zap"
)

func TestReconcileApp_MissingClusterSetsError(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)

	tmpDir, _ := os.MkdirTemp("", "reconcile-missing-cluster")
	defer os.RemoveAll(tmpDir)
	appCfg := filepath.Join(tmpDir, "apps.json")

	a := &app.Application{
		Name:            "a1",
		RepoURL:         "https://github.com/org/repo.git",
		Branch:          "main",
		Path:            "manifests",
		ClusterName:     "missing",
		PollingInterval: time.Minute,
		Status:          "Pending",
	}
	apps.Add(a)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl.wg.Add(1)
	ctrl.reconcileApp(ctx, a, appCfg, cancel, make(chan struct{}, 1))

	if a.Status != "Error" {
		t.Fatalf("expected Error status when cluster missing, got %s", a.Status)
	}
}

func TestReconcileApp_InvalidKubeconfigSetsError(t *testing.T) {
	logger := zap.NewNop()
	apps := app.NewApplications()
	clusters := cluster.NewClusters()
	ctrl := NewController(logger, apps, clusters)

	tmpDir, _ := os.MkdirTemp("", "reconcile-invalid-kube")
	defer os.RemoveAll(tmpDir)
	appCfg := filepath.Join(tmpDir, "apps.json")

	a := &app.Application{
		Name:            "a1",
		RepoURL:         "https://github.com/org/repo.git",
		Branch:          "main",
		Path:            "manifests",
		ClusterName:     "c1",
		PollingInterval: time.Minute,
		Status:          "Pending",
	}
	apps.Add(a)
	clusters.Add(&cluster.Cluster{Name: "c1", KubeconfigPath: "/no/such/kubeconfig"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl.wg.Add(1)
	ctrl.reconcileApp(ctx, a, appCfg, cancel, make(chan struct{}, 1))

	if a.Status != "Error" {
		t.Fatalf("expected Error status for invalid kubeconfig, got %s", a.Status)
	}
}
