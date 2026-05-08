package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveApplications(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "app-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpFile := filepath.Join(tmpDir, "apps.json")

	// 1. Initial load (should be empty but not error if file missing)
	apps, err := LoadApplications(tmpFile)
	if err != nil {
		t.Errorf("LoadApplications() error = %v", err)
	}
	if apps.Len() != 0 {
		t.Errorf("Expected 0 apps, got %d", apps.Len())
	}

	// 2. Add an app and save
	testApp := &Application{
		Name:     "test-app",
		RepoURL:  "https://github.com/user/repo.git",
		Status:   "Pending",
		Interval: "5m",
	}
	apps.Add(testApp)

	if err := SaveApplications(apps, tmpFile); err != nil {
		t.Errorf("SaveApplications() error = %v", err)
	}

	// 3. Load again and verify
	apps2, err := LoadApplications(tmpFile)
	if err != nil {
		t.Errorf("LoadApplications() (second load) error = %v", err)
	}
	if apps2.Len() != 1 {
		t.Errorf("Expected 1 app, got %d", apps2.Len())
	}

	loadedApp, exists := apps2.Get("test-app")
	if !exists {
		t.Fatal("App 'test-app' not found after reload")
	}
	if loadedApp.RepoURL != testApp.RepoURL {
		t.Errorf("Expected repo %s, got %s", testApp.RepoURL, loadedApp.RepoURL)
	}
}

func TestApplication_ToTable(t *testing.T) {
	a := &Application{
		Name:        "test",
		RepoURL:     "https://github.com/user/repo.git",
		ClusterName: "prod",
		Status:      "Synced",
	}

	row := a.ToTableRow(false)
	if len(row) != 6 {
		t.Errorf("Expected 6 columns in summary table, got %d", len(row))
	}

	rowDetails := a.ToTableRow(true)
	// Headers for details: NAME, REPO URL, BRANCH, PATH, CLUSTER, INTERVAL, STATUS, SYNC POLICY, LAST SYNCED HASH, FAILURES, MESSAGE
	if len(rowDetails) != 11 {
		t.Errorf("Expected 11 columns in details table, got %d", len(rowDetails))
	}
}
