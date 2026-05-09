package cluster

import (
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegisterRoutes_RegistersClusterEndpoints(t *testing.T) {
	e := echo.New()
	g := e.Group("/api/v1")
	h := &Handler{}

	RegisterRoutes(g, h)

	registered := map[string]bool{}
	for _, r := range e.Routes() {
		registered[r.Method+" "+r.Path] = true
	}

	expected := []string{
		"POST /api/v1/clusters",
		"GET /api/v1/clusters",
		"GET /api/v1/clusters/:name",
		"DELETE /api/v1/clusters/:name",
		"POST /api/v1/clusters/:name/check",
	}
	for _, route := range expected {
		if !registered[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}
