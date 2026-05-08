package common

import (
	"testing"
	"time"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		max    int
		expect string
	}{
		{"hello world", 5, "he..."},
		{"hello", 10, "hello"},
		{"hello world", 2, "he"},
		{"hello world", 11, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := TruncateString(tt.s, tt.max)
			if got != tt.expect {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.expect)
			}
		})
	}
}

func TestDefaultIfEmpty(t *testing.T) {
	tests := []struct {
		s      string
		def    string
		expect string
	}{
		{"", "default", "default"},
		{"  ", "default", "default"},
		{"value", "default", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := DefaultIfEmpty(tt.s, tt.def)
			if got != tt.expect {
				t.Errorf("DefaultIfEmpty(%q, %q) = %q, want %q", tt.s, tt.def, got, tt.expect)
			}
		})
	}
}

func TestGetRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		t      time.Time
		expect string
	}{
		{"zero", time.Time{}, "N/A"},
		{"now", now.Add(-10 * time.Second), "just now"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"hours", now.Add(-2 * time.Hour), "2h ago"},
		{"days", now.Add(-3 * 24 * time.Hour), "3d ago"},
		{"weeks", now.Add(-2 * 7 * 24 * time.Hour), "2w ago"},
		{"months", now.Add(-60 * 24 * time.Hour), "2mo ago"},
		{"years", now.Add(-400 * 24 * time.Hour), "13yr ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRelativeTime(tt.t)
			if got != tt.expect {
				t.Errorf("GetRelativeTime() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		isValid bool
	}{
		{"my-app", true},
		{"app123", true},
		{"", false},
		{"  ", false},
		{"a", true},
		{"-app", false},
		{"app-", false},
		{"too-long-" + string(make([]byte, 60)), false},
		{"invalid!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if (err == nil) != tt.isValid {
				t.Errorf("ValidateName(%q) error = %v, want valid=%v", tt.name, err, tt.isValid)
			}
		})
	}
}
