package cluster

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCluster_Rendering(t *testing.T) {
	c := &Cluster{
		Name:           "prod",
		KubeconfigPath: "/path/to/kube",
		Status:         "Active",
	}

	// 1. Table
	headers := c.ToTableHeaders(false)
	if len(headers) != 4 {
		t.Errorf("Expected 4 headers, got %d", len(headers))
	}
	row := c.ToTableRow(false)
	if row[1] != "✅ Active" {
		t.Errorf("Expected status '✅ Active', got %s", row[1])
	}

	// 2. JSON
	jsonMap := c.ToJSONMap()
	if jsonMap["name"] != "prod" {
		t.Error("JSON name mismatch")
	}

	// 3. YAML
	yaml := c.ToYAMLString()
	if !strings.Contains(yaml, "name: prod") {
		t.Error("YAML content mismatch")
	}
}

func TestFormatClusterStatus(t *testing.T) {
	tests := []struct {
		status string
		expect string
	}{
		{"Active", "✅ Active"},
		{"Error", "❗ Error"},
		{"Pending", "⏳ Pending"},
		{"Disconnected", "❌ Disconnected"},
		{"Unknown", "❓ Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := formatClusterStatus(tt.status)
			if got != tt.expect {
				t.Errorf("formatClusterStatus(%q) = %q, want %q", tt.status, got, tt.expect)
			}
		})
	}
}
