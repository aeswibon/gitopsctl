package app

import (
	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	"net/http"
	"testing"
)

func TestGet_ReturnsApplication(t *testing.T) {
	h, e, apps, _ := newTestHandler()
	apps.Add(&appcore.Application{Name: "a1"})

	c, rec := newJSONContext(e, http.MethodGet, "/applications/a1", "")
	c.SetParamNames("name")
	c.SetParamValues("a1")

	if err := h.Get(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	h, e, _, _ := newTestHandler()

	c, _ := newJSONContext(e, http.MethodGet, "/applications/missing", "")
	c.SetParamNames("name")
	c.SetParamValues("missing")

	err := h.Get(c)
	if err == nil {
		t.Fatal("expected 404 error")
	}
}
