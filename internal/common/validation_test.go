package common

import (
	"os"
	"testing"
)

func TestIsValidGitURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/user/repo.git", true},
		{"http://github.com/user/repo.git", true},
		{"git@github.com:user/repo.git", true},
		{"not-a-url", false},
		{"ftp://github.com/user/repo.git", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := IsValidGitURL(tt.url); got != tt.expected {
				t.Errorf("IsValidGitURL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestIsValidRepoPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"k8s/manifests", true},
		{"/k8s/manifests/", true},
		{"/", false},
		{"///", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsValidRepoPath(tt.path); got != tt.expected {
				t.Errorf("IsValidRepoPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://github.com/user/repo", false},
		{"git@github.com:user/repo.git", false},
		{"invalid-url", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, err := ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateKubeconfigFile(t *testing.T) {
	// 1. Valid file
	tmpFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	validConfig := `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`
	_ = os.WriteFile(tmpFile.Name(), []byte(validConfig), 0644)
	_ = tmpFile.Close()

	if err := ValidateKubeconfigFile(tmpFile.Name()); err != nil {
		t.Errorf("ValidateKubeconfigFile() error = %v, want nil", err)
	}

	// 2. Non-existent file
	if err := ValidateKubeconfigFile("/path/to/nothing"); err == nil {
		t.Error("Expected error for non-existent file")
	}

	// 3. Directory
	tmpDir, err := os.MkdirTemp("", "kubeconfig-dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := ValidateKubeconfigFile(tmpDir); err == nil {
		t.Error("Expected error for directory")
	}

	noClusters, err := os.CreateTemp("", "kubeconfig-empty-clusters")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(noClusters.Name()) }()
	emptyClustersYAML := `apiVersion: v1
kind: Config
clusters: []
contexts: []
current-context: ""
users: []
`
	if err := os.WriteFile(noClusters.Name(), []byte(emptyClustersYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateKubeconfigFile(noClusters.Name()); err == nil {
		t.Error("Expected error when kubeconfig defines no clusters")
	}
}
