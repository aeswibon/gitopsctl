package cluster

import (
	"testing"
	"time"

	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
)

func TestConvertToResponse_MapsFields(t *testing.T) {
	now := time.Now()
	src := &clustercore.Cluster{
		Name:              "c1",
		KubeconfigPath:    "/tmp/kubeconfig",
		RegisteredAt:      now,
		Status:            "Healthy",
		Message:           "ok",
		LastCheckedAt:     now,
		AllowedNamespaces: []string{"n1"},
	}

	got := ConvertToResponse(src)
	if got.Name != src.Name || got.KubeconfigPath != src.KubeconfigPath {
		t.Fatalf("response fields not mapped correctly: %+v", got)
	}
	if len(got.AllowedNamespaces) != 1 || got.AllowedNamespaces[0] != "n1" {
		t.Errorf("expected allowed namespaces [n1], got %v", got.AllowedNamespaces)
	}
}

func TestConvertToResponseList_MapsItems(t *testing.T) {
	srcs := []*clustercore.Cluster{
		{Name: "c1"},
		{Name: "c2"},
	}
	got := ConvertToResponseList(srcs)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}
