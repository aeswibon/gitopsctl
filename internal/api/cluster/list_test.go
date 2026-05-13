package cluster

import (
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"net/http"
	"testing"
)

func TestList_ReturnsClusters(t *testing.T) {
	h, e, _, clusters := newTestHandler()
	clusters.Add(&clustercore.Cluster{Name: "c1"})
	clusters.Add(&clustercore.Cluster{Name: "c2"})

	c, rec := newJSONContext(e, http.MethodGet, "/clusters", "")
	if err := h.List(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGet_ReturnsCluster(t *testing.T) {
	h, e, _, clusters := newTestHandler()
	clusters.Add(&clustercore.Cluster{Name: "c1"})

	c, rec := newJSONContext(e, http.MethodGet, "/clusters/c1", "")
	c.SetParamNames("name")
	c.SetParamValues("c1")

	if err := h.Get(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	h, e, _, _ := newTestHandler()

	c, _ := newJSONContext(e, http.MethodGet, "/clusters/missing", "")
	c.SetParamNames("name")
	c.SetParamValues("missing")

	err := h.Get(c)
	if err == nil {
		t.Fatal("expected 404 error")
	}
}
