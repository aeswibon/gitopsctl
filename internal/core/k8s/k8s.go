package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/sops"
	"aeswibon.com/github/gitopsctl/internal/metrics"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const (
	// DefaultAPITimeout is the default timeout for Kubernetes API requests
	DefaultAPITimeout = 30 * time.Second
	// DefaultQPS is the default queries per second for client-go
	DefaultQPS = 100
	// DefaultBurst is the default burst for client-go
	DefaultBurst = 100
)

// ResourceMetadata stores the basic identity of a Kubernetes resource.
type ResourceMetadata struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
}

// ClientSet holds Kubernetes clients for dynamic interactions.
// It encapsulates the dynamic client, REST mapper, and configuration required
// for interacting with Kubernetes resources.
type ClientSet struct {
	// logger is used for logging operations and errors.
	logger *zap.Logger
	// kubeconfigPath is the path to the kubeconfig file used for authentication.
	kubeconfigPath string
	// dynamicClient is the Kubernetes dynamic client for interacting with arbitrary resources.
	dynamicClient dynamic.Interface
	// mapper is the REST mapper for translating GroupVersionKind to REST resources.
	mapper meta.RESTMapper
	// config is the Kubernetes configuration used to initialize clients.
	config *rest.Config
	// allowedNamespaces is a list of namespaces this client is allowed to interact with.
	allowedNamespaces []string
	// defaultNamespace is the namespace to use if none is specified, or to enforce.
	defaultNamespace string
	// enforceNamespace if true, all namespaced resources will be forced into defaultNamespace.
	enforceNamespace bool
}

// NewClientSet initializes a Kubernetes client set.
// It attempts to use the provided kubeconfig file to build the configuration.
// If the kubeconfig file is not provided or fails, it falls back to in-cluster configuration.
func NewClientSet(logger *zap.Logger, kubeconfigPath string, allowedNamespaces []string, defaultNamespace string, enforceNamespace bool) (*ClientSet, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
		logger.Info("No kubeconfig path provided, attempting to use default", zap.String("path", kubeconfigPath))
	}

	// Use the specified kubeconfig file to build the config
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		// Fallback to in-cluster config if kubeconfig is not found or fails
		logger.Warn("Failed to build config from kubeconfig, attempting in-cluster config", zap.Error(err))
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("could not build Kubernetes config from kubeconfig (%s) or in-cluster: %w", kubeconfigPath, err)
		}
		logger.Info("Using in-cluster configuration")
	} else {
		logger.Info("Using kubeconfig", zap.String("path", kubeconfigPath))
	}

	config.Timeout = DefaultAPITimeout
	config.QPS = DefaultQPS
	config.Burst = DefaultBurst

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	return &ClientSet{
		logger:            logger,
		kubeconfigPath:    kubeconfigPath,
		dynamicClient:     dynamicClient,
		mapper:            mapper,
		config:            config,
		allowedNamespaces: allowedNamespaces,
		defaultNamespace:  defaultNamespace,
		enforceNamespace:  enforceNamespace,
	}, nil
}

// ApplyManifests applies Kubernetes manifests from a given directory to the cluster.
// It checks if the directory contains a Helm chart or Kustomization file and builds it if present.
// Otherwise, it processes all YAML files in the specified directory.
func (cs *ClientSet) ApplyManifests(ctx context.Context, manifestsDir string, appName, clusterName string, createNamespace bool, previouslyApplied []ResourceMetadata, prune bool) ([]ResourceMetadata, []error) {
	cs.logger.Info("Applying manifests", zap.String("directory", manifestsDir))
	var appliedResources []ResourceMetadata
	var applyErrors []error

	// Pre-process: Decrypt any SOPS-encrypted files in the directory
	if err := cs.decryptDirectory(manifestsDir); err != nil {
		cs.logger.Warn("Failed to decrypt some files, proceeding anyway", zap.Error(err))
	}

	if hasHelmChart(manifestsDir) {
		cs.logger.Info("Detected Helm Chart, rendering with helm", zap.String("directory", manifestsDir))

		actionConfig := new(action.Configuration)
		actionConfig.Log = func(format string, v ...interface{}) {
			cs.logger.Debug(fmt.Sprintf(format, v...))
		}

		client := action.NewInstall(actionConfig)
		client.DryRun = true
		client.ClientOnly = true
		client.ReleaseName = "gitopsctl-release"
		client.Namespace = cs.defaultNamespace
		if client.Namespace == "" {
			client.Namespace = "default"
		}

		chartReq, err := loader.Load(manifestsDir)
		if err != nil {
			cs.logger.Error("Failed to load Helm chart", zap.Error(err))
			return nil, []error{fmt.Errorf("failed to load Helm chart: %w", err)}
		}

		rel, err := client.Run(chartReq, nil)
		if err != nil {
			cs.logger.Error("Helm template rendering failed", zap.Error(err))
			return nil, []error{fmt.Errorf("helm template rendering failed: %w", err)}
		}

		cs.logger.Debug("Successfully rendered Helm chart")
		return cs.applyYAMLData(ctx, []byte(rel.Manifest), filepath.Join(manifestsDir, "Chart.yaml"), appName, clusterName, false) // Helm usually handles NS
	}

	fSys := filesys.MakeFsOnDisk()
	if hasKustomization(fSys, manifestsDir) {
		cs.logger.Info("Detected Kustomization, building with kustomize", zap.String("directory", manifestsDir))
		k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
		resMap, err := k.Run(fSys, manifestsDir)
		if err != nil {
			cs.logger.Error("Kustomize build failed", zap.Error(err))
			return nil, []error{fmt.Errorf("kustomize build failed: %w", err)}
		}

		yamlBytes, err := resMap.AsYaml()
		if err != nil {
			cs.logger.Error("Failed to convert kustomize output to YAML", zap.Error(err))
			return nil, []error{fmt.Errorf("failed to convert kustomize output to yaml: %w", err)}
		}

		cs.logger.Debug("Successfully built kustomization, applying generated YAML")
		return cs.applyYAMLData(ctx, yamlBytes, filepath.Join(manifestsDir, "kustomization"), appName, clusterName, false)
	}

	cs.logger.Info("No kustomization found, applying raw YAML files", zap.String("directory", manifestsDir))
	err := filepath.WalkDir(manifestsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			applyErrors = append(applyErrors, fmt.Errorf("filesystem error walking %s: %w", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}

		cs.logger.Debug("Processing manifest file", zap.String("file", path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			cs.logger.Error("Failed to read manifest file", zap.String("file", path), zap.Error(readErr))
			applyErrors = append(applyErrors, fmt.Errorf("failed to read file %s: %w", path, readErr))
			return nil
		}

		fileResources, fileErrors := cs.applyYAMLData(ctx, data, path, appName, clusterName, createNamespace) // Use the flag here
		if len(fileErrors) > 0 {
			applyErrors = append(applyErrors, fileErrors...)
		}
		appliedResources = append(appliedResources, fileResources...)
		return nil
	})
	if err != nil {
		applyErrors = append(applyErrors, fmt.Errorf("error during manifest directory walk %s: %w", manifestsDir, err))
	}

	if prune && len(applyErrors) == 0 {
		pruneErrors := cs.pruneResources(ctx, previouslyApplied, appliedResources, appName, clusterName)
		if len(pruneErrors) > 0 {
			applyErrors = append(applyErrors, pruneErrors...)
		}
	}

	return appliedResources, applyErrors
}

func (cs *ClientSet) pruneResources(ctx context.Context, previouslyApplied, currentlyApplied []ResourceMetadata, appName, clusterName string) []error {
	var pruneErrors []error

	// Map of currently applied for fast lookup
	currentMap := make(map[string]bool)
	for _, r := range currentlyApplied {
		key := fmt.Sprintf("%s/%s/%s/%s", r.Group, r.Kind, r.Namespace, r.Name)
		currentMap[key] = true
	}

	for _, r := range previouslyApplied {
		key := fmt.Sprintf("%s/%s/%s/%s", r.Group, r.Kind, r.Namespace, r.Name)
		if !currentMap[key] {
			cs.logger.Info("Pruning resource",
				zap.String("kind", r.Kind),
				zap.String("name", r.Name),
				zap.String("namespace", r.Namespace))

			gvk := schema.GroupVersionKind{
				Group:   r.Group,
				Version: r.Version,
				Kind:    r.Kind,
			}
			mapping, err := cs.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				pruneErrors = append(pruneErrors, fmt.Errorf("failed to get REST mapping for pruning %s: %w", r.Kind, err))
				continue
			}

			var dr dynamic.ResourceInterface
			if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
				dr = cs.dynamicClient.Resource(mapping.Resource).Namespace(r.Namespace)
			} else {
				dr = cs.dynamicClient.Resource(mapping.Resource)
			}

			deleteErr := dr.Delete(ctx, r.Name, metav1.DeleteOptions{})
			if deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
				cs.logger.Error("Failed to prune resource",
					zap.String("kind", r.Kind),
					zap.String("name", r.Name),
					zap.Error(deleteErr))
				pruneErrors = append(pruneErrors, fmt.Errorf("failed to prune %s %s/%s: %w", r.Kind, r.Namespace, r.Name, deleteErr))
			} else {
				metrics.K8sApplyTotal.WithLabelValues(appName, clusterName, r.Kind, "pruned").Inc()
			}
		}
	}

	return pruneErrors
}

func (cs *ClientSet) decryptDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") && !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		decrypted, wasEncrypted, err := sops.Decrypt(path)
		if err != nil {
			return err
		}

		if wasEncrypted {
			cs.logger.Debug("Decrypted SOPS file", zap.String("file", path))
			return os.WriteFile(path, decrypted, 0644)
		}
		return nil
	})
}

func hasKustomization(fSys filesys.FileSystem, dir string) bool {
	for _, name := range []string{"kustomization.yaml", "kustomization.yml", "Kustomization"} {
		if fSys.Exists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func hasHelmChart(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "Chart.yaml"))
	if err == nil && !info.IsDir() {
		return true
	}
	info, err = os.Stat(filepath.Join(dir, "Chart.yml"))
	if err == nil && !info.IsDir() {
		return true
	}
	return false
}

// applyYAMLData takes a byte slice of YAML documents (separated by ---) and applies them to the cluster.
func (cs *ClientSet) applyYAMLData(ctx context.Context, data []byte, sourceName, appName, clusterName string, createNamespace bool) ([]ResourceMetadata, []error) {
	var appliedResources []ResourceMetadata
	var applyErrors []error
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	objects := strings.Split(string(data), "\n---")

	for i, objStr := range objects {
		trimmedObjStr := strings.TrimSpace(objStr)
		if trimmedObjStr == "" {
			continue
		}

		unstructuredObj := &unstructured.Unstructured{}
		_, gvk, decodeErr := decoder.Decode([]byte(trimmedObjStr), nil, unstructuredObj)
		if decodeErr != nil {
			cs.logger.Error("Failed to decode YAML object", zap.String("source", sourceName), zap.Int("documentIdx", i), zap.Error(decodeErr))
			applyErrors = append(applyErrors, fmt.Errorf("failed to decode YAML from %s (doc %d): %w", sourceName, i, decodeErr))
			continue
		}

		if unstructuredObj.GetName() == "" {
			cs.logger.Warn("Skipping unnamed resource in manifest", zap.String("source", sourceName), zap.Int("documentIdx", i), zap.String("kind", gvk.Kind))
			applyErrors = append(applyErrors, fmt.Errorf("skipping unnamed resource in %s (doc %d) of kind %s", sourceName, i, gvk.Kind))
			continue
		}

		mapping, mappingErr := cs.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if mappingErr != nil {
			cs.logger.Error("Failed to get REST mapping for GVK",
				zap.String("gvk", gvk.String()), zap.String("source", sourceName), zap.Error(mappingErr))
			applyErrors = append(applyErrors, fmt.Errorf("failed to get REST mapping for %s in %s: %w", gvk.String(), sourceName, mappingErr))
			continue
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := unstructuredObj.GetNamespace()

			// Handle Namespace Enforcement or Defaulting
			if cs.enforceNamespace && cs.defaultNamespace != "" {
				if ns != "" && ns != cs.defaultNamespace {
					cs.logger.Debug("Enforcing namespace override",
						zap.String("original", ns),
						zap.String("enforced", cs.defaultNamespace),
						zap.String("kind", gvk.Kind),
						zap.String("name", unstructuredObj.GetName()))
				}
				ns = cs.defaultNamespace
				unstructuredObj.SetNamespace(ns)
			} else if ns == "" {
				ns = cs.defaultNamespace
				if ns == "" {
					ns = "default"
				}
				unstructuredObj.SetNamespace(ns)
				cs.logger.Debug("Namespace not specified, using default",
					zap.String("default", ns),
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()))
			}

			// Security Check: Namespace Restriction
			if !cs.isNamespaceAllowed(ns) {
				cs.logger.Error("Namespace not allowed", zap.String("namespace", ns), zap.String("kind", gvk.Kind), zap.String("name", unstructuredObj.GetName()))
				applyErrors = append(applyErrors, fmt.Errorf("namespace %q is not in the allowed list for this cluster", ns))
				continue
			}

			if createNamespace {
				if err := cs.ensureNamespace(ctx, ns); err != nil {
					cs.logger.Error("Failed to ensure namespace", zap.String("namespace", ns), zap.Error(err))
					applyErrors = append(applyErrors, fmt.Errorf("failed to ensure namespace %q: %w", ns, err))
					continue
				}
			}

			dr = cs.dynamicClient.Resource(mapping.Resource).Namespace(ns)
		} else {
			// cluster-scoped resources should not specify the namespace
			dr = cs.dynamicClient.Resource(mapping.Resource)
		}

		// Try to get the resource
		existing, getErr := dr.Get(ctx, unstructuredObj.GetName(), metav1.GetOptions{})

		if getErr == nil {
			// Check for drift before updating
			if cs.isDrifted(existing, unstructuredObj) {
				cs.logger.Info("Drift detected, resolving...",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()))
				metrics.AppDriftTotal.WithLabelValues(appName, clusterName, gvk.Kind).Inc()
			}
		}

		if getErr != nil {
			// Resource does not exist, create it
			_, createErr := dr.Create(ctx, unstructuredObj, metav1.CreateOptions{})
			if createErr != nil {
				metrics.K8sApplyTotal.WithLabelValues(appName, clusterName, gvk.Kind, "failure").Inc()
				cs.logger.Error("Failed to create resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()),
					zap.Error(createErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to create %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), sourceName, createErr))
				continue
			}
			metrics.K8sApplyTotal.WithLabelValues(appName, clusterName, gvk.Kind, "success").Inc()
			cs.logger.Info("Created resource",
				zap.String("kind", gvk.Kind),
				zap.String("name", unstructuredObj.GetName()),
				zap.String("namespace", unstructuredObj.GetNamespace()))
		} else {
			// Resource exists, update it (using simple update for MVP)
			_, updateErr := dr.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
			if updateErr != nil {
				metrics.K8sApplyTotal.WithLabelValues(appName, clusterName, gvk.Kind, "failure").Inc()
				cs.logger.Error("Failed to update resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()),
					zap.Error(updateErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to update %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), sourceName, updateErr))
				continue
			}
			metrics.K8sApplyTotal.WithLabelValues(appName, clusterName, gvk.Kind, "success").Inc()
			cs.logger.Info("Updated resource",
				zap.String("kind", gvk.Kind),
				zap.String("name", unstructuredObj.GetName()),
				zap.String("namespace", unstructuredObj.GetNamespace()))
		}

		appliedResources = append(appliedResources, ResourceMetadata{
			Group:     gvk.Group,
			Version:   gvk.Version,
			Kind:      gvk.Kind,
			Namespace: unstructuredObj.GetNamespace(),
			Name:      unstructuredObj.GetName(),
		})
	}
	return appliedResources, applyErrors
}

// CheckConnectivity verifies connectivity to the Kubernetes cluster.
// It uses the Kubernetes clientset to fetch the server version, ensuring the cluster is reachable.
// CheckConnectivity verifies connectivity to the Kubernetes cluster.
// It uses the Kubernetes clientset to fetch the server version, ensuring the cluster is reachable.
func (cs *ClientSet) CheckConnectivity(ctx context.Context) error {
	if cs.config == nil {
		return fmt.Errorf("failed to create kubernetes clientset: missing Kubernetes config")
	}
	kubeClient, err := kubernetes.NewForConfig(cs.config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes server version: %w", err)
	}
	return nil
}

// GetResourceHealth checks the health of a specific Kubernetes resource.
// Returns status (Healthy, Progressing, Degraded, Unknown) and a descriptive message.
func (cs *ClientSet) GetResourceHealth(ctx context.Context, r ResourceMetadata) (string, string, error) {
	gvk := schema.GroupVersionKind{Group: r.Group, Version: r.Version, Kind: r.Kind}
	mapping, err := cs.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "Unknown", fmt.Sprintf("Failed to get mapping: %v", err), err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// Security Check: Namespace Restriction
		if !cs.isNamespaceAllowed(r.Namespace) {
			return "Unknown", fmt.Sprintf("Namespace %q is not allowed", r.Namespace), fmt.Errorf("namespace %q is not in the allowed list", r.Namespace)
		}
		dr = cs.dynamicClient.Resource(mapping.Resource).Namespace(r.Namespace)
	} else {
		dr = cs.dynamicClient.Resource(mapping.Resource)
	}

	obj, err := dr.Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		return "Unknown", fmt.Sprintf("Failed to get resource: %v", err), err
	}

	// Basic health assessment logic based on resource kind
	switch r.Kind {
	case "Deployment":
		status, ok, _ := unstructured.NestedMap(obj.Object, "status")
		if !ok {
			return "Progressing", "Waiting for status", nil
		}
		replicas, _, _ := unstructured.NestedInt64(status, "replicas")
		readyReplicas, _, _ := unstructured.NestedInt64(status, "readyReplicas")
		updatedReplicas, _, _ := unstructured.NestedInt64(status, "updatedReplicas")
		availableReplicas, _, _ := unstructured.NestedInt64(status, "availableReplicas")

		if availableReplicas >= replicas && readyReplicas >= replicas && updatedReplicas >= replicas {
			return "Healthy", fmt.Sprintf("%d/%d replicas available", availableReplicas, replicas), nil
		}
		return "Progressing", fmt.Sprintf("%d/%d replicas available", availableReplicas, replicas), nil

	case "Service":
		return "Healthy", "Service created", nil

	case "Pod":
		status, ok, _ := unstructured.NestedMap(obj.Object, "status")
		if !ok {
			return "Progressing", "Waiting for status", nil
		}
		phase, _, _ := unstructured.NestedString(status, "phase")
		if phase == "Running" || phase == "Succeeded" {
			return "Healthy", "Pod is " + phase, nil
		}
		if phase == "Failed" {
			return "Degraded", "Pod failed", nil
		}
		return "Progressing", "Pod is " + phase, nil

	default:
		return "Healthy", "Resource applied", nil
	}
}

func (cs *ClientSet) isNamespaceAllowed(ns string) bool {
	if len(cs.allowedNamespaces) == 0 {
		return true
	}
	for _, allowed := range cs.allowedNamespaces {
		if allowed == ns {
			return true
		}
	}
	return false
}

func (cs *ClientSet) ensureNamespace(ctx context.Context, ns string) error {
	if ns == "default" || ns == "" {
		return nil
	}

	kubeClient, err := kubernetes.NewForConfig(cs.config)
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	cs.logger.Info("Creating namespace", zap.String("namespace", ns))
	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}, metav1.CreateOptions{})
	return err
}

func (cs *ClientSet) isDrifted(existing, target *unstructured.Unstructured) bool {
	// Simple drift detection: compare 'spec' and 'data' and 'labels' and 'annotations'
	// ignoring system fields like status, resourceVersion, etc.

	fieldsToCompare := []string{"spec", "data", "stringData", "labels", "annotations"}
	for _, field := range fieldsToCompare {
		existingVal, ok1, _ := unstructured.NestedFieldNoCopy(existing.Object, field)
		targetVal, ok2, _ := unstructured.NestedFieldNoCopy(target.Object, field)

		if !ok1 && !ok2 {
			continue
		}
		if ok1 != ok2 {
			return true
		}

		// Deep equal comparison (simplified for MVP)
		existingJSON, _ := json.Marshal(existingVal)
		targetJSON, _ := json.Marshal(targetVal)
		if string(existingJSON) != string(targetJSON) {
			return true
		}
	}
	return false
}
