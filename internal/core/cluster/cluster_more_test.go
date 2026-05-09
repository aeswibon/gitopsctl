package cluster

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClusters_LockListDeleteAndLen(t *testing.T) {
	cs := NewClusters()
	cs.Lock()
	cs.Add(&Cluster{Name: "c1"})
	cs.Unlock()

	cs.RLock()
	if len(cs.List()) != 1 || cs.Len() != 1 {
		t.Fatalf("expected one cluster after add")
	}
	cs.RUnlock()

	cs.Lock()
	cs.Delete("c1")
	cs.Unlock()
	if cs.Len() != 0 {
		t.Fatalf("expected zero clusters after delete")
	}
}

func TestVerifyCluster(t *testing.T) {
	origWd, _ := os.Getwd()
	tmpDir, _ := os.MkdirTemp("", "verify-cluster")
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(origWd)
	_ = os.Chdir(tmpDir)

	cs := NewClusters()
	cs.Add(&Cluster{Name: "c1", KubeconfigPath: "/tmp/kube", RegisteredAt: time.Now()})
	if err := SaveClusters(cs, DefaultClusterConfigFile); err != nil {
		t.Fatalf("save clusters failed: %v", err)
	}

	cl, ok, err := VerifyCluster("c1")
	if err != nil || !ok || cl == nil {
		t.Fatalf("VerifyCluster expected success, got cl=%v ok=%v err=%v", cl, ok, err)
	}
}

func TestClusterFormattingHelpers(t *testing.T) {
	now := time.Now()
	c := &Cluster{
		Name:           "c1",
		Status:         "Active",
		KubeconfigPath: "/tmp/kubeconfig",
		Message:        "all good",
		RegisteredAt:   now,
		LastCheckedAt:  now,
	}
	if len(c.ToTableHeaders(false)) == 0 || len(c.ToTableRow(true)) == 0 {
		t.Fatal("expected table formatting helpers to return values")
	}
	if c.ToJSONMap()["name"] != "c1" {
		t.Fatal("expected json map to include cluster name")
	}
	if y := c.ToYAMLString(); y == "" {
		t.Fatal("expected yaml string output")
	}

	// exercise format statuses
	_ = formatClusterStatus("active")
	_ = formatClusterStatus("error")
	_ = formatClusterStatus("pending")
	_ = formatClusterStatus("unknown")

	_ = filepath.Separator
}
