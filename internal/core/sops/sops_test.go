package sops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{
			name:     "encrypted yaml",
			data:     "apiVersion: v1\nkind: Secret\nsops:\n  version: 3.7.1\n  mac: 123",
			expected: true,
		},
		{
			name:     "plain yaml",
			data:     "apiVersion: v1\nkind: Secret\ndata:\n  foo: bar",
			expected: false,
		},
		{
			name:     "empty data",
			data:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEncrypted([]byte(tt.data)); got != tt.expected {
				t.Errorf("isEncrypted() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDecrypt_PlainFile(t *testing.T) {
	// Create a temp plain file
	tmpDir, err := os.MkdirTemp("", "sops-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpFile := filepath.Join(tmpDir, "plain.yaml")
	content := "foo: bar"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Decrypt(tmpFile)
	if err != nil {
		t.Errorf("Decrypt() error = %v", err)
	}
	if string(got) != content {
		t.Errorf("Decrypt() got = %s, want %s", string(got), content)
	}
}
