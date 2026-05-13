package cluster

import (
	"net/http"
	"testing"
)

func TestRegister_CreatesCluster(t *testing.T) {
	h, e, _, clusters := newTestHandler()

	body := `{"name":"c1","kubeconfig_path":"/tmp/kubeconfig","default_namespace":"default","allowed_namespaces":["a1","a2"]}`
	c, rec := newJSONContext(e, http.MethodPost, "/clusters", body)
	if err := h.Register(c); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if clusters.Len() != 1 {
		t.Fatalf("expected one cluster, got %d", clusters.Len())
	}

	cl, _ := clusters.Get("c1")
	if cl.DefaultNamespace != "default" {
		t.Errorf("expected default namespace 'default', got %s", cl.DefaultNamespace)
	}
	if len(cl.AllowedNamespaces) != 2 {
		t.Errorf("expected 2 allowed namespaces, got %d", len(cl.AllowedNamespaces))
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	h, e, _, _ := newTestHandler()

	// Missing name
	body := `{"kubeconfig_path":"/tmp/kubeconfig"}`
	c, _ := newJSONContext(e, http.MethodPost, "/clusters", body)
	if err := h.Register(c); err == nil {
		t.Fatal("expected error for missing name")
	}

	// Missing kubeconfig_path
	body = `{"name":"c1"}`
	c, _ = newJSONContext(e, http.MethodPost, "/clusters", body)
	if err := h.Register(c); err == nil {
		t.Fatal("expected error for missing kubeconfig_path")
	}
}
