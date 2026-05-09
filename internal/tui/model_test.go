package tui

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func sampleApp(name string) AppResponse {
	return AppResponse{
		Name:                name,
		RepoURL:             "https://example.com/" + name + ".git",
		Branch:              "main",
		Path:                "manifests",
		ClusterName:         "prod",
		Interval:            "1m",
		LastSyncedGitHash:   "1234567890abcdef",
		Status:              "Synced",
		Message:             "all good",
		ConsecutiveFailures: 2,
	}
}

func sampleCluster(name string) ClusterResponse {
	now := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	return ClusterResponse{
		Name:           name,
		KubeconfigPath: "/tmp/" + name + ".yaml",
		RegisteredAt:   now,
		Status:         "Active",
		Message:        "ready",
		LastCheckedAt:  now,
	}
}

func updateModel(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	got, _ := m.Update(msg)
	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", got)
	}
	return model
}

func TestNewModelAndInit(t *testing.T) {
	m := NewModel("http://example.test")
	if !m.loading {
		t.Fatal("expected model to start loading")
	}
	if m.client == nil || m.client.baseURL != "http://example.test" {
		t.Fatalf("unexpected client %#v", m.client)
	}
	if m.ctx == nil || m.cancel == nil {
		t.Fatal("expected context and cancel function")
	}
	if cmd := m.Init(); cmd == nil {
		t.Fatal("expected Init to return a command")
	}
	m.cancel()
}

func TestModelFetchCommands(t *testing.T) {
	m := NewModel("http://gitopsctl.test")
	m.client = clientWithTransport(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/applications":
			return response(http.StatusOK, `[{"name":"frontend"}]`), nil
		case "/api/v1/clusters":
			return response(http.StatusOK, `[{"name":"prod"}]`), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})

	if msg := m.fetchApps()(); len(msg.(appsLoadedMsg)) != 1 {
		t.Fatalf("expected one app, got %#v", msg)
	}
	if msg := m.fetchClusters()(); len(msg.(clustersLoadedMsg)) != 1 {
		t.Fatalf("expected one cluster, got %#v", msg)
	}

	m.client = clientWithTransport(func(r *http.Request) (*http.Response, error) {
		return response(http.StatusInternalServerError, "nope"), nil
	})
	if msg := m.fetchApps()(); msg == nil {
		t.Fatal("expected app fetch error")
	} else if _, ok := msg.(errorMsg); !ok {
		t.Fatalf("expected errorMsg, got %#v", msg)
	}
	if msg := m.fetchClusters()(); msg == nil {
		t.Fatal("expected cluster fetch error")
	} else if _, ok := msg.(errorMsg); !ok {
		t.Fatalf("expected errorMsg, got %#v", msg)
	}
}

func TestModelUpdateLoadsDataAndClampsCursors(t *testing.T) {
	m := NewModel("http://example.test")
	m.appCursor = 5
	m.clusterCursor = 5

	m = updateModel(t, m, appsLoadedMsg{sampleApp("frontend")})
	if m.loading || m.err != nil || m.appCursor != 0 || len(m.apps) != 1 {
		t.Fatalf("unexpected apps state: loading=%v err=%v cursor=%d apps=%d", m.loading, m.err, m.appCursor, len(m.apps))
	}

	m = updateModel(t, m, clustersLoadedMsg{sampleCluster("prod")})
	if m.loading || m.err != nil || m.clusterCursor != 0 || len(m.clusters) != 1 {
		t.Fatalf("unexpected clusters state: loading=%v err=%v cursor=%d clusters=%d", m.loading, m.err, m.clusterCursor, len(m.clusters))
	}
}

func TestModelUpdateErrorAndExpiredStatus(t *testing.T) {
	m := NewModel("http://example.test")
	m.statusMsg = "done"
	m.statusUntil = time.Now().Add(-time.Second)

	m = updateModel(t, m, errorMsg(errors.New("boom")))
	if m.loading || m.err == nil || m.statusMsg != "" || !m.statusUntil.IsZero() {
		t.Fatalf("unexpected error state: loading=%v err=%v status=%q until=%v", m.loading, m.err, m.statusMsg, m.statusUntil)
	}
}

func TestModelKeyboardNavigationAndRefresh(t *testing.T) {
	m := NewModel("http://example.test")
	m.apps = []AppResponse{sampleApp("a"), sampleApp("b")}
	m.clusters = []ClusterResponse{sampleCluster("c1"), sampleCluster("c2")}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.appCursor != 1 {
		t.Fatalf("expected app cursor 1, got %d", m.appCursor)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.appCursor != 0 {
		t.Fatalf("expected app cursor 0, got %d", m.appCursor)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.state != clustersView {
		t.Fatalf("expected clusters view, got %v", m.state)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.clusterCursor != 1 {
		t.Fatalf("expected cluster cursor 1, got %d", m.clusterCursor)
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.state != appsView {
		t.Fatalf("expected apps view, got %v", m.state)
	}

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = got.(Model)
	if !m.loading || cmd == nil {
		t.Fatal("expected refresh to set loading and return command")
	}
}

func TestModelConfirmationActions(t *testing.T) {
	var seen []string
	client := clientWithTransport(func(r *http.Request) (*http.Response, error) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		return response(http.StatusNoContent, ""), nil
	})

	tests := []struct {
		name    string
		state   viewState
		key     rune
		wantReq string
	}{
		{name: "sync app", state: appsView, key: 's', wantReq: "POST /api/v1/applications/frontend/sync"},
		{name: "unregister app", state: appsView, key: 'u', wantReq: "DELETE /api/v1/applications/frontend"},
		{name: "check cluster", state: clustersView, key: 'c', wantReq: "POST /api/v1/clusters/prod/check"},
		{name: "unregister cluster", state: clustersView, key: 'u', wantReq: "DELETE /api/v1/clusters/prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen = nil
			m := NewModel("http://gitopsctl.test")
			m.client = client
			m.apps = []AppResponse{sampleApp("frontend")}
			m.clusters = []ClusterResponse{sampleCluster("prod")}
			m.state = tt.state

			m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})
			if m.confirmMsg == "" || m.confirmAction == nil {
				t.Fatal("expected confirmation prompt")
			}
			m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			if m.confirmMsg != "" || m.confirmAction != nil || m.statusMsg == "" {
				t.Fatalf("confirmation was not cleared: msg=%q actionIsNil=%v status=%q", m.confirmMsg, m.confirmAction == nil, m.statusMsg)
			}
			if strings.Join(seen, "\n") != tt.wantReq {
				t.Fatalf("unexpected request: want %q got %v", tt.wantReq, seen)
			}
		})
	}
}

func TestModelConfirmationCancelAndQuit(t *testing.T) {
	m := NewModel("http://example.test")
	m.confirmMsg = "Really?"
	m.confirmAction = func() { t.Fatal("confirm action should not run") }

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.confirmMsg != "" || m.confirmAction != nil {
		t.Fatal("expected confirmation to clear")
	}

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = got.(Model)
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	select {
	case <-m.ctx.Done():
	default:
		t.Fatal("expected context to be canceled")
	}
}

func TestModelWindowSizeAndView(t *testing.T) {
	m := NewModel("http://example.test")
	if view := m.View(); view != "" {
		t.Fatalf("expected empty view before sizing, got %q", view)
	}

	m = updateModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m.apps = []AppResponse{sampleApp("frontend")}
	m.clusters = []ClusterResponse{sampleCluster("prod")}
	m.err = errors.New("temporary")
	m.confirmMsg = "Sync?"

	view := m.View()
	for _, want := range []string{"GitOpsCTL", "Applications", "frontend", "temporary", "Sync?", "sync", "unregister"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected view to contain %q\n%s", want, view)
		}
	}

	m.state = clustersView
	view = m.View()
	for _, want := range []string{"Clusters", "prod", "check", "unregister"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected cluster view to contain %q\n%s", want, view)
		}
	}
}

func TestRenderHelpers(t *testing.T) {
	m := NewModel("http://example.test")
	if got := m.renderAppList(40, 10); !strings.Contains(got, "No applications") {
		t.Fatalf("unexpected empty app list %q", got)
	}
	if got := m.renderClusterList(40, 10); !strings.Contains(got, "No clusters") {
		t.Fatalf("unexpected empty cluster list %q", got)
	}
	if got := m.renderAppDetail(40); !strings.Contains(got, "Select an application") {
		t.Fatalf("unexpected empty app detail %q", got)
	}
	if got := m.renderClusterDetail(40); !strings.Contains(got, "Select a cluster") {
		t.Fatalf("unexpected empty cluster detail %q", got)
	}

	m.apps = []AppResponse{sampleApp("frontend")}
	m.apps[0].ClusterName = ""
	m.apps[0].LastSyncedGitHash = ""
	m.apps[0].Message = "this message is long enough to be truncated"
	if got := m.renderAppDetail(25); !strings.Contains(got, "frontend") || !strings.Contains(got, "Failures") {
		t.Fatalf("unexpected app detail %q", got)
	}

	m.clusters = []ClusterResponse{sampleCluster("prod")}
	m.clusters[0].LastCheckedAt = time.Time{}
	m.clusters[0].Message = ""
	if got := m.renderClusterDetail(40); !strings.Contains(got, "prod") || !strings.Contains(got, "Last Checked") {
		t.Fatalf("unexpected cluster detail %q", got)
	}

	for _, status := range []string{"Synced", "Active", "Error", "Unreachable", "Pending", "Syncing", "OutOfSync", "", "Other"} {
		if got := StatusChip(status); got == "" {
			t.Fatalf("StatusChip(%q) returned empty string", status)
		}
	}
	if max(1, 2) != 2 || max(3, 2) != 3 {
		t.Fatal("max returned unexpected result")
	}
	if got := kv("A", "B"); !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Fatalf("unexpected kv output %q", got)
	}
}
