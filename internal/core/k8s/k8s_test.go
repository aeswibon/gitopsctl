package k8s

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func fakeClientSet() *ClientSet {
	configMapGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	configMapGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	namespaceGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}
	namespaceGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{{Group: "", Version: "v1"}})
	mapper.AddSpecific(configMapGVK, configMapGVR, configMapGVR, meta.RESTScopeNamespace)
	mapper.AddSpecific(namespaceGVK, namespaceGVR, namespaceGVR, meta.RESTScopeRoot)

	return &ClientSet{
		logger:        zap.NewNop(),
		dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		mapper:        mapper,
	}
}

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
	_, errors := cs.applyYAMLData(context.Background(), []byte(""), "test", "app1", "cluster1", false)
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors for empty data, got %d", len(errors))
	}

	// Test with invalid YAML
	_, errors = cs.applyYAMLData(context.Background(), []byte("invalid: yaml: :"), "test", "app1", "cluster1", false)
	if len(errors) == 0 {
		t.Error("Expected error for invalid YAML, got 0")
	}
}

func TestNewClientSet_EmptyPath(t *testing.T) {
	logger := zap.NewNop()
	// This will try to load from home dir, might fail but we check if it handles empty path
	_, _ = NewClientSet(logger, "", nil, "", false)
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
	_, errs := cs.ApplyManifests(context.Background(), tmpDir, "app1", "cluster1", false, nil, false)
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

	cs, err := NewClientSet(zap.NewNop(), kcfg, nil, "", false)
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
	_, errs := cs.applyYAMLData(context.Background(), yamlData, "inline", "app1", "cluster1", false)
	if len(errs) == 0 {
		t.Fatal("expected error for unnamed resource")
	}
}

func TestApplyYAMLData_CreateAndUpdateNamespacedResource(t *testing.T) {
	cs := fakeClientSet()
	ctx := context.Background()
	first := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: sample
data:
  key: first
`)

	if _, errs := cs.applyYAMLData(ctx, first, "inline", "app1", "cluster1", false); len(errs) != 0 {
		t.Fatalf("expected create to succeed, got %v", errs)
	}

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	got, err := cs.dynamicClient.Resource(gvr).Namespace("default").Get(ctx, "sample", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected created configmap: %v", err)
	}
	if got.GetNamespace() != "default" {
		t.Fatalf("expected default namespace, got %q", got.GetNamespace())
	}

	second := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: sample
  namespace: default
data:
  key: second
`)
	if _, errs := cs.applyYAMLData(ctx, second, "inline", "app1", "cluster1", false); len(errs) != 0 {
		t.Fatalf("expected update to succeed, got %v", errs)
	}

	got, err = cs.dynamicClient.Resource(gvr).Namespace("default").Get(ctx, "sample", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected updated configmap: %v", err)
	}
	if got.Object["data"].(map[string]interface{})["key"] != "second" {
		t.Fatalf("expected updated data, got %#v", got.Object["data"])
	}
}

func TestApplyYAMLData_ClusterScopedResource(t *testing.T) {
	cs := fakeClientSet()
	yamlData := []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: staging
`)
	if _, errs := cs.applyYAMLData(context.Background(), yamlData, "inline", "app1", "cluster1", false); len(errs) != 0 {
		t.Fatalf("expected namespace apply to succeed, got %v", errs)
	}
}

func TestApplyManifests_RawYamlSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply-raw-success")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("skip me"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cm.yml"), []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: raw
data:
  key: value
`), 0644); err != nil {
		t.Fatal(err)
	}

	if _, errs := fakeClientSet().ApplyManifests(context.Background(), tmpDir, "app1", "cluster1", false, nil, false); len(errs) != 0 {
		t.Fatalf("expected raw manifest apply to succeed, got %v", errs)
	}
}

func TestApplyManifests_KustomizeSuccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply-kustomize-success")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte("resources:\n- cm.yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cm.yaml"), []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: from-kustomize
data:
  key: value
`), 0644); err != nil {
		t.Fatal(err)
	}

	if _, errs := fakeClientSet().ApplyManifests(context.Background(), tmpDir, "app1", "cluster1", false, nil, false); len(errs) != 0 {
		t.Fatalf("expected kustomize apply to succeed, got %v", errs)
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
	_, _ = cs.ApplyManifests(context.Background(), tmpDir, "app1", "cluster1", false, nil, false)
}

func TestApplyManifests_KustomizeBuildFailure(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "apply-kustomize-fail")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := os.WriteFile(filepath.Join(tmpDir, "kustomization.yaml"), []byte("resources:\n- missing.yaml\n"), 0644); err != nil {
		t.Fatalf("failed writing kustomization file: %v", err)
	}
	cs := &ClientSet{logger: zap.NewNop()}
	_, errs := cs.ApplyManifests(context.Background(), tmpDir, "app1", "cluster1", false, nil, false)
	if len(errs) == 0 {
		t.Fatal("expected kustomize build error")
	}
}

func TestNamespaceRestriction(t *testing.T) {
	cs := fakeClientSet()
	cs.allowedNamespaces = []string{"allowed"}

	// 1. applyYAMLData with disallowed namespace
	badYaml := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: sample
  namespace: forbidden
`)
	_, errs := cs.applyYAMLData(context.Background(), badYaml, "test", "app1", "cluster1", false)
	if len(errs) == 0 {
		t.Fatal("expected error for forbidden namespace in applyYAMLData")
	}
	if !strings.Contains(errs[0].Error(), "not in the allowed list") {
		t.Fatalf("unexpected error message: %v", errs[0])
	}

	// 2. applyYAMLData with allowed namespace
	goodYaml := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: sample
  namespace: allowed
`)
	_, errs = cs.applyYAMLData(context.Background(), goodYaml, "test", "app1", "cluster1", false)
	if len(errs) != 0 {
		t.Fatalf("expected success for allowed namespace, got %v", errs)
	}

	// 3. GetResourceHealth with disallowed namespace
	_, _, err := cs.GetResourceHealth(context.Background(), ResourceMetadata{
		Namespace: "forbidden",
		Name:      "sample",
		Kind:      "ConfigMap",
	})
	if err == nil {
		t.Fatal("expected error for forbidden namespace in GetResourceHealth")
	}

	// 4. GetResourceHealth with allowed namespace
	_, _, err = cs.GetResourceHealth(context.Background(), ResourceMetadata{
		Namespace: "allowed",
		Name:      "sample",
		Kind:      "ConfigMap",
	})
	if err != nil {
		t.Fatalf("expected success for allowed namespace in GetResourceHealth, got %v", err)
	}
}
