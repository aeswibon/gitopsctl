package cluster

import (
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"net/http"
	"testing"
)

func TestUnregister_RemovesCluster(t *testing.T) {
	h, e, _, clusters := newTestHandler()
	clusters.Add(&clustercore.Cluster{Name: "c1"})

	c, rec := newJSONContext(e, http.MethodDelete, "/clusters/c1", "")
	c.SetParamNames("name")
	c.SetParamValues("c1")

	if err := h.Unregister(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if clusters.Len() != 0 {
		t.Fatal("expected cluster to be removed")
	}
}

func TestUnregister_NotFound(t *testing.T) {
	h, e, _, _ := newTestHandler()

	c, _ := newJSONContext(e, http.MethodDelete, "/clusters/missing", "")
	c.SetParamNames("name")
	c.SetParamValues("missing")

	if err := h.Unregister(c); err == nil {
		t.Fatal("expected error for missing cluster")
	}
}
