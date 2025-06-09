package common

import (
	"fmt"
	"strings"
	"time"
)

// TruncateString truncates a string to a maximum length and appends "..." if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// DefaultIfEmpty returns the default value if the string is empty or just whitespace.
func DefaultIfEmpty(s, defaultVal string) string {
	if strings.TrimSpace(s) == "" {
		return defaultVal
	}
	return s
}

// GetRelativeTime formats a time.Time object into a human-readable relative time string.
func GetRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "N/A" // or "never"
	}

	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(diff.Hours()/24/7))
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(diff.Hours()/24/30))
	default:
		return fmt.Sprintf("%dyr ago", int(diff.Hours()/24/30))
	}
}

func isAlphaNumeric(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
}

// ValidateName checks if the provided name is valid in kubernetes naming conventions.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name is required (use --name or -n flag)")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty or whitespace only")
	}

	if len(name) > 63 {
		return fmt.Errorf("cluster name too long (maximum 63 characters)")
	}

	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return fmt.Errorf("cluster name must start and end with an alphanumeric character")
	}

	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return fmt.Errorf("invalid character '%c' in cluster name '%s'; only alphanumeric characters and hyphens are allowed", char, name)
		}
	}

	return nil
}

// ConfirmAction prompts the user for confirmation before proceeding with an action.
func ConfirmAction(message string) bool {
	fmt.Printf("%s [y/N]: ", message)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
