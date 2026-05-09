package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApproveAppCommand_HTTP202(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/applications/myapp/approve") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`approved`))
	}))
	defer ts.Close()

	prevURL := apiBaseURL
	apiBaseURL = ts.URL
	defer func() { apiBaseURL = prevURL }()

	root := RootCmd()
	_, err := executeCommand(root, "approve-app", "-n", "myapp", "-c", "deadbeef")
	if err != nil {
		t.Fatalf("approve-app: %v", err)
	}
}

func TestSyncAppCommand_HTTP202(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/applications/myapp/sync") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`sync ok`))
	}))
	defer ts.Close()

	prevURL := apiBaseURL
	apiBaseURL = ts.URL
	defer func() { apiBaseURL = prevURL }()

	root := RootCmd()
	_, err := executeCommand(root, "sync-app", "-n", "myapp")
	if err != nil {
		t.Fatalf("sync-app: %v", err)
	}
}
