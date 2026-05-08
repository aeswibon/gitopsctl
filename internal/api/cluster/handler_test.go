package cluster

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aeswibon.com/github/gitopsctl/internal/controller"
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestHandler_Register(t *testing.T) {
	e := echo.New()
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := controller.NewController(logger, apps, clusters)

	h := NewHandler(logger, clusters, apps, ctrl)

	reqBody := `{"name":"test-cluster","kubeconfig_path":"/path/to/kubeconfig"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Register(c); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if clusters.Len() != 1 {
		t.Error("Cluster was not registered")
	}
}

func TestHandler_List(t *testing.T) {
	e := echo.New()
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := controller.NewController(logger, apps, clusters)

	clusters.Add(&clustercore.Cluster{Name: "c1"})
	h := NewHandler(logger, clusters, apps, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.List(c); err != nil {
		t.Errorf("List() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestHandler_Unregister(t *testing.T) {
	e := echo.New()
	logger := zap.NewNop()
	apps := appcore.NewApplications()
	clusters := clustercore.NewClusters()
	ctrl := controller.NewController(logger, apps, clusters)

	clusters.Add(&clustercore.Cluster{Name: "c1"})
	h := NewHandler(logger, clusters, apps, ctrl)

	req := httptest.NewRequest(http.MethodDelete, "/clusters/c1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("name")
	c.SetParamValues("c1")

	if err := h.Unregister(c); err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	if clusters.Len() != 0 {
		t.Error("Cluster was not unregistered")
	}
}
