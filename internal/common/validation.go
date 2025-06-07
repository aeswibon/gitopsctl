package common

import (
	"fmt"
	"net/url"
	"strings"
)

// IsValidGitURL validates if a string is a basic Git URL (HTTPS or SSH format)
//
// It checks for common patterns like "git@host:repo.git" for SSH and "http(s)://host/repo.git" for HTTPS.
func IsValidGitURL(s string) bool {
	if strings.HasPrefix(s, "git@") && strings.Contains(s, ":") {
		// Basic check for SSH format: git@host:repo/path.git
		return true
	}
	if u, err := url.ParseRequestURI(s); err == nil {
		// Basic check for HTTPS format
		return u.Scheme == "http" || u.Scheme == "https"
	}
	return false
}

// IsValidRepoPath validates if a string is a valid repository path
//
// It checks that the path is not empty or just slashes after trimming leading and trailing slashes.
// This is useful to ensure that the path provided for manifests in the repository is meaningful.
func IsValidRepoPath(s string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSuffix(s, "/"), "/")
	return trimmed != "" // Path cannot be empty or just slashes after trimming
}

// ParseURL is a helper to parse a URL. Using net/url.ParseRequestURI for stricter parsing.
//
// It ensures that the URL has a scheme and host, which is important for Git URLs.
func ParseURL(rawurl string) (*url.URL, error) {
	u, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return nil, err
	}
	// Also ensure scheme is present for a valid URL
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("URL missing scheme or host")
	}
	return u, nil
}
