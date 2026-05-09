package sops

import (
	"bytes"
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

func TestDecryptReader_Plain(t *testing.T) {
	content := "foo: bar"
	reader := bytes.NewReader([]byte(content))

	got, err := DecryptReader(reader, "yaml")
	if err != nil {
		t.Errorf("DecryptReader() error = %v", err)
	}
	if string(got) != content {
		t.Errorf("DecryptReader() got = %s, want %s", string(got), content)
	}
}

func TestDecrypt_FileNotFound(t *testing.T) {
	_, err := Decrypt("non-existent-file")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestDecrypt_EncryptedInvalidDataReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sops-invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpFile := filepath.Join(tmpDir, "secret.yaml")
	encryptedLike := "sops:\n  version: 3.7.1\n  mac: deadbeef\nfoo: ENC[AES256_GCM,data:bad]\n"
	if err := os.WriteFile(tmpFile, []byte(encryptedLike), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Decrypt(tmpFile); err == nil {
		t.Fatal("expected decrypt error for invalid encrypted payload")
	}
}

func TestDecryptReader_EncryptedInvalidDataReturnsError(t *testing.T) {
	encryptedLike := "sops:\n  version: 3.7.1\n  mac: deadbeef\nfoo: ENC[AES256_GCM,data:bad]\n"
	reader := bytes.NewReader([]byte(encryptedLike))
	if _, err := DecryptReader(reader, "yaml"); err == nil {
		t.Fatal("expected decrypt reader error for invalid encrypted payload")
	}
}
