package cluster

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveClusters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cluster-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpFile := filepath.Join(tmpDir, "clusters.json")

	// 1. Initial load
	clusters, err := LoadClusters(tmpFile)
	if err != nil {
		t.Errorf("LoadClusters() error = %v", err)
	}
	if clusters.Len() != 0 {
		t.Errorf("Expected 0 clusters, got %d", clusters.Len())
	}

	// 2. Add a cluster and save
	testCluster := &Cluster{
		Name:           "prod",
		KubeconfigPath: "/path/to/kubeconfig",
		Status:         "Active",
	}
	clusters.Add(testCluster)

	if err := SaveClusters(clusters, tmpFile); err != nil {
		t.Errorf("SaveClusters() error = %v", err)
	}

	// 3. Load again and verify
	clusters2, err := LoadClusters(tmpFile)
	if err != nil {
		t.Errorf("LoadClusters() (second load) error = %v", err)
	}
	if clusters2.Len() != 1 {
		t.Errorf("Expected 1 cluster, got %d", clusters2.Len())
	}

	loaded, exists := clusters2.Get("prod")
	if !exists {
		t.Fatal("Cluster 'prod' not found after reload")
	}
	if loaded.KubeconfigPath != testCluster.KubeconfigPath {
		t.Errorf("Expected kubeconfig %s, got %s", testCluster.KubeconfigPath, loaded.KubeconfigPath)
	}
}
