package common

import "testing"

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
