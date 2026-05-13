package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	// 4. Test List
	list := apps2.List()
	if len(list) != 1 {
		t.Errorf("Expected list length 1, got %d", len(list))
	}

	// 5. Test Delete
	apps2.Delete("test-app")
	if apps2.Len() != 0 {
		t.Error("Delete failed")
	}

	// 6. Test Locking (simple call)
	apps2.Lock()
	_ = apps2.Len()
	apps2.Unlock()
	apps2.RLock()
	_ = apps2.Len()
	apps2.RUnlock()

}

func TestApplication_MoreRendering(t *testing.T) {
	a := &Application{
		Name:    "test",
		RepoURL: "https://github.com/user/repo.git",
	}

	headers := a.ToTableHeaders(false)
	if len(headers) != 6 {
		t.Errorf("Expected 6 headers, got %d", len(headers))
	}

	jsonMap := a.ToJSONMap()
	if jsonMap["name"] != "test" {
		t.Error("JSON name mismatch")
	}

	yaml := a.ToYAMLString()
	if !strings.Contains(yaml, "name: test") {
		t.Error("YAML content mismatch")
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

func TestIsSyncAllowed(t *testing.T) {
	// 2026-05-11 is a Monday
	baseTime := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		windows       []SyncWindow
		now           time.Time
		expectAllowed bool
	}{
		{
			name:          "No windows",
			windows:       nil,
			now:           baseTime,
			expectAllowed: true,
		},
		{
			name: "In allowed window",
			windows: []SyncWindow{
				{Start: "10:00", End: "14:00", Days: []string{"Monday"}, TimeZone: "UTC"},
			},
			now:           baseTime,
			expectAllowed: true,
		},
		{
			name: "Outside allowed window (time)",
			windows: []SyncWindow{
				{Start: "14:00", End: "16:00", Days: []string{"Monday"}, TimeZone: "UTC"},
			},
			now:           baseTime,
			expectAllowed: false,
		},
		{
			name: "Outside allowed window (day)",
			windows: []SyncWindow{
				{Start: "10:00", End: "14:00", Days: []string{"Tuesday"}, TimeZone: "UTC"},
			},
			now:           baseTime,
			expectAllowed: false,
		},
		{
			name: "In blackout window",
			windows: []SyncWindow{
				{Start: "10:00", End: "14:00", Deny: true, TimeZone: "UTC"},
			},
			now:           baseTime,
			expectAllowed: false,
		},
		{
			name: "Allowed window but also blackout window",
			windows: []SyncWindow{
				{Start: "08:00", End: "17:00", Deny: false, TimeZone: "UTC"},
				{Start: "11:00", End: "13:00", Deny: true, TimeZone: "UTC"},
			},
			now:           baseTime,
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Application{SyncWindows: tt.windows}
			allowed, _ := a.IsSyncAllowed(tt.now)
			if allowed != tt.expectAllowed {
				t.Errorf("IsSyncAllowed() = %v, want %v", allowed, tt.expectAllowed)
			}
		})
	}
}

func TestLoadApplications_Durations(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "app-dur-test")
	defer func() { _ = os.RemoveAll(tmpDir) }()
	tmpFile := filepath.Join(tmpDir, "apps.json")

	content := `[
		{
			"name": "dur-app",
			"interval": "10m",
			"initial_backoff": "1s",
			"max_backoff": "1h"
		}
	]`
	_ = os.WriteFile(tmpFile, []byte(content), 0644)

	apps, err := LoadApplications(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	a, _ := apps.Get("dur-app")
	if a.PollingInterval != 10*time.Minute {
		t.Errorf("Expected 10m, got %v", a.PollingInterval)
	}
	if a.RetryBackoff != time.Second {
		t.Errorf("Expected 1s, got %v", a.RetryBackoff)
	}
	if a.MaxRetryBackoff != time.Hour {
		t.Errorf("Expected 1h, got %v", a.MaxRetryBackoff)
	}
}
