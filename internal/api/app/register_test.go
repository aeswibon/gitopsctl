package app

import (
	"net/http"
	"testing"

	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
)

func TestRegister_CreatesApplication(t *testing.T) {
	h, e, apps, clusters := newTestHandler()
	clusters.Add(&clustercore.Cluster{Name: "c1"})

	c, rec := newJSONContext(e, http.MethodPost, "/applications", `{"name":"a1","repo_url":"https://github.com/org/repo.git","branch":"main","path":"/manifests/","cluster_name":"c1","interval":"1m"}`)
	if err := h.Register(c); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if apps.Len() != 1 {
		t.Fatalf("expected one app to be registered, got %d", apps.Len())
	}
}

func TestRegister_WithPhase7Fields(t *testing.T) {
	h, e, apps, clusters := newTestHandler()
	clusters.Add(&clustercore.Cluster{Name: "c1"})

	payload := `{
		"name":"a1",
		"repo_url":"https://github.com/org/repo.git",
		"branch":"main",
		"path":"/manifests/",
		"cluster_name":"c1",
		"interval":"1m",
		"max_retries": 5,
		"prune": true,
		"depends_on": ["dep1"],
		"sync_windows": [
			{"start": "09:00", "end": "17:00", "deny": false}
		]
	}`
	c, rec := newJSONContext(e, http.MethodPost, "/applications", payload)
	if err := h.Register(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	a, _ := apps.Get("a1")
	if a.MaxRetries != 5 {
		t.Errorf("expected 5 retries, got %d", a.MaxRetries)
	}
	if !a.Prune {
		t.Error("expected prune to be true")
	}
	if len(a.DependsOn) != 1 || a.DependsOn[0] != "dep1" {
		t.Errorf("expected [dep1], got %v", a.DependsOn)
	}
	if len(a.SyncWindows) != 1 {
		t.Error("expected 1 sync window")
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	h, e, _, _ := newTestHandler()

	// Missing name
	c, _ := newJSONContext(e, http.MethodPost, "/applications", `{"repo_url":"https://github.com/org/repo.git","interval":"1m"}`)
	err := h.Register(c)
	if err == nil {
		t.Error("expected 400 for missing name, got nil error")
	}

	// Invalid URL
	c, _ = newJSONContext(e, http.MethodPost, "/applications", `{"name":"a1","repo_url":"not-a-url","interval":"1m"}`)
	err = h.Register(c)
	if err == nil {
		t.Error("expected 400 for invalid URL, got nil error")
	}
}
