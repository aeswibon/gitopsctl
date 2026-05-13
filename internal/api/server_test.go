package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aeswibon.com/github/gitopsctl/internal/controller"
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"aeswibon.com/github/gitopsctl/internal/events"
	"go.uber.org/zap"
)

func TestServer_HealthCheck(t *testing.T) {
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := &controller.Controller{} // Mock/minimal controller
	server := NewServer(logger, apps, clusters, ctrl, nil, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := server.Echo().NewContext(req, rec)

	if err := server.HealthCheck(c); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %q", rec.Body.String())
	}
}

func TestServer_Metrics(t *testing.T) {
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := &controller.Controller{}
	server := NewServer(logger, apps, clusters, ctrl, nil, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestServer_GetEventHistory(t *testing.T) {
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := &controller.Controller{}
	history := events.NewHistorySink(10)
	server := NewServer(logger, apps, clusters, ctrl, nil, history, "")

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	c := server.Echo().NewContext(req, rec)

	if err := server.GetEventHistory(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestServer_APIKeyMiddleware(t *testing.T) {
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := &controller.Controller{}
	server := NewServer(logger, apps, clusters, ctrl, nil, nil, "secret-key")

	// 1. No key - should fail
	req := httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// 2. Correct key - should pass
	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec = httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestServer_StreamEvents_Nil(t *testing.T) {
	server := NewServer(zap.NewNop(), appcore.NewApplications(), clustercore.NewClusters(), nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	c := server.Echo().NewContext(req, rec)
	if err := server.StreamEvents(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rec.Code)
	}
}

func TestServer_GetEventHistory_Nil(t *testing.T) {
	server := NewServer(zap.NewNop(), appcore.NewApplications(), clustercore.NewClusters(), nil, nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/events/history", nil)
	rec := httptest.NewRecorder()
	c := server.Echo().NewContext(req, rec)
	if err := server.GetEventHistory(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rec.Code)
	}
}
