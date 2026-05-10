package sops

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/getsops/sops/v3/decrypt"
)

// Decrypt decrypts a SOPS-encrypted file and returns the plaintext data and a boolean indicating if it was decrypted.
// If the file is not encrypted (does not contain SOPS metadata), it returns the original content and false.
func Decrypt(filePath string) ([]byte, bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Simple check if the file is likely SOPS encrypted
	if !isEncrypted(data) {
		return data, false, nil
	}

	format := "yaml"
	if strings.HasSuffix(filePath, ".json") {
		format = "json"
	}

	// Decrypt the file
	decrypted, err := decrypt.Data(data, format)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decrypt file %s: %w", filePath, err)
	}

	return decrypted, true, nil
}

// isEncrypted returns true if the data appears to be a SOPS-encrypted file.
func isEncrypted(data []byte) bool {
	content := string(data)
	return strings.Contains(content, "sops:") && (strings.Contains(content, "version:") || strings.Contains(content, "mac:") || strings.Contains(content, "lastmodified:"))
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
