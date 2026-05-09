package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func TestHasKustomization(t *testing.T) {
	fSys := filesys.MakeFsInMemory()
	dir := "/test"
	_ = fSys.Mkdir(dir)

	if hasKustomization(fSys, dir) {
		t.Error("Expected false for empty dir")
	}

	_ = fSys.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(""))

	if !hasKustomization(fSys, dir) {
		t.Error("Expected true for kustomization.yaml")
	}
}

func TestHasHelmChart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "helm-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if hasHelmChart(tmpDir) {
		t.Error("Expected false for empty dir")
	}

	_ = os.WriteFile(filepath.Join(tmpDir, "Chart.yaml"), []byte(""), 0644)

	if !hasHelmChart(tmpDir) {
		t.Error("Expected true for Chart.yaml")
	}
}

func TestApplyYAMLData_Validation(t *testing.T) {
	logger := zap.NewNop()
	cs := &ClientSet{
		logger: logger,
	}

	// Test with empty data
	errors := cs.applyYAMLData(context.Background(), []byte(""), "test")
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors for empty data, got %d", len(errors))
	}

	// Test with invalid YAML
	errors = cs.applyYAMLData(context.Background(), []byte("invalid: yaml: :"), "test")
	if len(errors) == 0 {
		t.Error("Expected error for invalid YAML, got 0")
	}
}

func TestNewClientSet_EmptyPath(t *testing.T) {
	logger := zap.NewNop()
	// This will try to load from home dir, might fail but we check if it handles empty path
	_, _ = NewClientSet(logger, "")
}

func TestHasHelmChart_YmlVariant(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "helm-test-yml")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "Chart.yml"), []byte("apiVersion: v2"), 0644); err != nil {
		t.Fatalf("failed writing chart: %v", err)
	}
	if !hasHelmChart(tmpDir) {
		t.Fatal("expected true for Chart.yml")
	}
}

func TestDecryptDirectory_NonEncryptedYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "decrypt-dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := filepath.Join(tmpDir, "app.yaml")
	if err := os.WriteFile(p, []byte("kind: ConfigMap\nmetadata:\n  name: sample\n"), 0644); err != nil {
		t.Fatalf("failed writing file: %v", err)
	}

	cs := &ClientSet{logger: zap.NewNop()}
	if err := cs.decryptDirectory(tmpDir); err != nil {
		t.Fatalf("decryptDirectory returned unexpected error: %v", err)
	}
}

func TestApplyManifests_RawYamlWithDecodeError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply-manifests")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Invalid YAML should be surfaced as apply errors without requiring cluster clients.
	if err := os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte("invalid: yaml: :"), 0644); err != nil {
		t.Fatalf("failed writing bad manifest: %v", err)
	}

	cs := &ClientSet{logger: zap.NewNop()}
	errs := cs.ApplyManifests(context.Background(), tmpDir)
	if len(errs) == 0 {
		t.Fatal("expected at least one apply error for invalid YAML")
	}
}

func TestCheckConnectivity_NilConfig(t *testing.T) {
	cs := &ClientSet{logger: zap.NewNop(), config: nil}
	if err := cs.CheckConnectivity(context.Background()); err == nil {
		t.Fatal("expected error when config is nil")
	}
}

func TestCheckConnectivity_ServerUnreachable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-conn")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	kcfg := filepath.Join(tmpDir, "kube.yaml")
	kubeYAML := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:1
    insecure-skip-tls-verify: true
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
users:
- name: default
  user: {}
`
	if err := os.WriteFile(kcfg, []byte(kubeYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := NewClientSet(zap.NewNop(), kcfg)
	if err != nil {
		t.Fatalf("NewClientSet: %v", err)
	}
	if err := cs.CheckConnectivity(context.Background()); err == nil {
		t.Fatal("expected connectivity error for unreachable API server")
	}
}

func TestApplyYAMLData_UnnamedResource(t *testing.T) {
	cs := &ClientSet{logger: zap.NewNop()}
	yamlData := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: default
data:
  key: value
`)
	errs := cs.applyYAMLData(context.Background(), yamlData, "inline")
	if len(errs) == 0 {
		t.Fatal("expected error for unnamed resource")
	}
}

func TestApplyManifests_HelmLoadFailure(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "apply-helm-fail")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create marker Chart file but invalid chart structure.
	if err := os.WriteFile(filepath.Join(tmpDir, "Chart.yaml"), []byte("apiVersion: v2\nname: broken\nversion: 0.1.0\n"), 0644); err != nil {
		t.Fatalf("failed writing chart: %v", err)
	}
	cs := &ClientSet{logger: zap.NewNop()}
	_ = cs.ApplyManifests(context.Background(), tmpDir)
}

func TestApplyManifests_KustomizeBuildFailure(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "apply-kustomize-fail")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte("resources:\n- missing.yaml\n"), 0644); err != nil {
		t.Fatalf("failed writing kustomization file: %v", err)
	}
	cs := &ClientSet{logger: zap.NewNop()}
	errs := cs.ApplyManifests(context.Background(), tmpDir)
	if len(errs) == 0 {
		t.Fatal("expected kustomize build error")
	}
}
