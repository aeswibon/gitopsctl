package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAndNormalizeInput_SetsDefaults(t *testing.T) {
	appName = " my-app "
	repoURL = "https://github.com/example/repo.git"
	branch = ""
	pathInRepo = "/deploy/"
	clusterName = " prod "
	interval = ""
	syncPolicy = ""
	webhookURL = " https://example.com/hook "
	webhookSecret = " secret "

	cfg, err := validateAndNormalizeInput()
	if err != nil {
		t.Fatalf("validateAndNormalizeInput() unexpected error: %v", err)
	}
	if cfg.branch != "main" {
		t.Fatalf("expected default branch main, got %q", cfg.branch)
	}
	if cfg.interval != "5m" {
		t.Fatalf("expected default interval 5m, got %q", cfg.interval)
	}
	if cfg.pathInRepo != "deploy" {
		t.Fatalf("expected normalized path deploy, got %q", cfg.pathInRepo)
	}
	if cfg.syncPolicy != "auto" {
		t.Fatalf("expected default sync policy auto, got %q", cfg.syncPolicy)
	}
	if cfg.pollingInterval != 5*time.Minute {
		t.Fatalf("expected parsed polling interval 5m, got %s", cfg.pollingInterval)
	}
}

func TestValidateAndNormalizeInput_RejectsBadValues(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		appName = ""
		repoURL = ""
		pathInRepo = ""
		clusterName = ""
		interval = "1m"
		syncPolicy = "auto"

		_, err := validateAndNormalizeInput()
		if err == nil || !strings.Contains(err.Error(), "missing required fields") {
			t.Fatalf("expected missing fields error, got %v", err)
		}
	})

	t.Run("invalid url", func(t *testing.T) {
		appName = "my-app"
		repoURL = "notaurl"
		pathInRepo = "deploy"
		clusterName = "prod"
		interval = "1m"
		syncPolicy = "auto"

		_, err := validateAndNormalizeInput()
		if err == nil || !strings.Contains(err.Error(), "invalid repository URL format") {
			t.Fatalf("expected invalid repository url error, got %v", err)
		}
	})

	t.Run("invalid sync policy", func(t *testing.T) {
		appName = "my-app"
		repoURL = "https://github.com/example/repo.git"
		pathInRepo = "deploy"
		clusterName = "prod"
		interval = "1m"
		syncPolicy = "weekly"

		_, err := validateAndNormalizeInput()
		if err == nil || !strings.Contains(err.Error(), "invalid sync policy") {
			t.Fatalf("expected invalid sync policy error, got %v", err)
		}
	})
}
