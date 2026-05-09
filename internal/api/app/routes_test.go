package app

import (
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegisterRoutes_RegistersExpectedPaths(t *testing.T) {
	e := echo.New()
	g := e.Group("/api/v1")
	RegisterRoutes(g, &Handler{})

	routes := map[string]bool{}
	for _, r := range e.Routes() {
		routes[r.Method+" "+r.Path] = true
	}

	expected := []string{
		"POST /api/v1/applications",
		"GET /api/v1/applications",
		"GET /api/v1/applications/:name",
		"DELETE /api/v1/applications/:name",
		"POST /api/v1/applications/:name/sync",
		"POST /api/v1/applications/:name/approve",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}
