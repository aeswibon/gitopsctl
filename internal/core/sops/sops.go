package sops

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/getsops/sops/v3/decrypt"
)

// Decrypt decrypts a SOPS-encrypted file and returns the plaintext data.
// If the file is not encrypted (does not contain SOPS metadata), it returns the original content.
func Decrypt(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Simple check if the file is likely SOPS encrypted
	if !isEncrypted(data) {
		return data, nil
	}

	// Decrypt the file
	decrypted, err := decrypt.Data(data, "yaml")
	if err != nil {
		// If it's not YAML, try JSON
		if strings.HasSuffix(filePath, ".json") {
			decrypted, err = decrypt.Data(data, "json")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt file %s: %w", filePath, err)
		}
	}

	return decrypted, nil
}

// isEncrypted returns true if the data appears to be a SOPS-encrypted file.
func isEncrypted(data []byte) bool {
	content := string(data)
	return strings.Contains(content, "sops:") && (strings.Contains(content, "version:") || strings.Contains(content, "mac:"))
}

// DecryptReader decrypts data from a reader.
func DecryptReader(r io.Reader, format string) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if !isEncrypted(data) {
		return data, nil
	}

	return decrypt.Data(data, format)
}
