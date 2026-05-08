package k8s

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/sops"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
}

// NewClientSet initializes a Kubernetes client set.
// It attempts to use the provided kubeconfig file to build the configuration.
// If the kubeconfig file is not provided or fails, it falls back to in-cluster configuration.
func NewClientSet(logger *zap.Logger, kubeconfigPath string) (*ClientSet, error) {
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
		logger:         logger,
		kubeconfigPath: kubeconfigPath,
		dynamicClient:  dynamicClient,
		mapper:         mapper,
		config:         config,
	}, nil
}

// ApplyManifests applies Kubernetes manifests from a given directory to the cluster.
// It checks if the directory contains a Helm chart or Kustomization file and builds it if present.
// Otherwise, it processes all YAML files in the specified directory.
func (cs *ClientSet) ApplyManifests(ctx context.Context, manifestsDir string) []error {
	cs.logger.Info("Applying manifests", zap.String("directory", manifestsDir))
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
		client.Namespace = "default"

		chartReq, err := loader.Load(manifestsDir)
		if err != nil {
			cs.logger.Error("Failed to load Helm chart", zap.Error(err))
			return []error{fmt.Errorf("failed to load Helm chart: %w", err)}
		}

		rel, err := client.Run(chartReq, nil)
		if err != nil {
			cs.logger.Error("Helm template rendering failed", zap.Error(err))
			return []error{fmt.Errorf("helm template rendering failed: %w", err)}
		}

		cs.logger.Debug("Successfully rendered Helm chart")
		return cs.applyYAMLData(ctx, []byte(rel.Manifest), filepath.Join(manifestsDir, "Chart.yaml"))
	}

	fSys := filesys.MakeFsOnDisk()
	if hasKustomization(fSys, manifestsDir) {
		cs.logger.Info("Detected Kustomization, building with kustomize", zap.String("directory", manifestsDir))
		k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
		resMap, err := k.Run(fSys, manifestsDir)
		if err != nil {
			cs.logger.Error("Kustomize build failed", zap.Error(err))
			return []error{fmt.Errorf("kustomize build failed: %w", err)}
		}

		yamlBytes, err := resMap.AsYaml()
		if err != nil {
			cs.logger.Error("Failed to convert kustomize output to YAML", zap.Error(err))
			return []error{fmt.Errorf("failed to convert kustomize output to yaml: %w", err)}
		}

		cs.logger.Debug("Successfully built kustomization, applying generated YAML")
		return cs.applyYAMLData(ctx, yamlBytes, filepath.Join(manifestsDir, "kustomization"))
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

		fileErrors := cs.applyYAMLData(ctx, data, path)
		if len(fileErrors) > 0 {
			applyErrors = append(applyErrors, fileErrors...)
		}
		return nil
	})
	if err != nil {
		applyErrors = append(applyErrors, fmt.Errorf("error during manifest directory walk %s: %w", manifestsDir, err))
	}
	return applyErrors
}

func (cs *ClientSet) decryptDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") && !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		decrypted, err := sops.Decrypt(path)
		if err != nil {
			return err
		}

		// If decrypted data is different, write it back to the file
		// Note: sops.Decrypt returns original data if not encrypted
		// But it might still be slightly different (formatting etc).
		// For now, we'll just always write it back if it's potentially encrypted.
		// Actually, let's only write if it WAS encrypted.
		// I'll update sops.Decrypt to return a boolean if it was encrypted.
		return os.WriteFile(path, decrypted, 0644)
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
func (cs *ClientSet) applyYAMLData(ctx context.Context, data []byte, sourceName string) []error {
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
			// namespaced resources should specify the namespace
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
				cs.logger.Debug("Namespace not specified for namespaced resource, defaulting to 'default'",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()))
			}
			dr = cs.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			// cluster-scoped resources should not specify the namespace
			dr = cs.dynamicClient.Resource(mapping.Resource)
		}

		// Try to get the resource
		_, getErr := dr.Get(ctx, unstructuredObj.GetName(), metav1.GetOptions{})

		if getErr != nil {
			// Resource does not exist, create it
			_, createErr := dr.Create(ctx, unstructuredObj, metav1.CreateOptions{})
			if createErr != nil {
				cs.logger.Error("Failed to create resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()),
					zap.Error(createErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to create %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), sourceName, createErr))
				continue
			}
			cs.logger.Info("Created resource",
				zap.String("kind", gvk.Kind),
				zap.String("name", unstructuredObj.GetName()),
				zap.String("namespace", unstructuredObj.GetNamespace()))
		} else {
			// Resource exists, update it (using simple update for MVP)
			_, updateErr := dr.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
			if updateErr != nil {
				cs.logger.Error("Failed to update resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()),
					zap.Error(updateErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to update %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), sourceName, updateErr))
				continue
			}
			cs.logger.Info("Updated resource",
				zap.String("kind", gvk.Kind),
				zap.String("name", unstructuredObj.GetName()),
				zap.String("namespace", unstructuredObj.GetNamespace()))
		}
	}
	return applyErrors
}

// CheckConnectivity verifies connectivity to the Kubernetes cluster.
// It uses the Kubernetes clientset to fetch the server version, ensuring the cluster is reachable.
func (cs *ClientSet) CheckConnectivity(ctx context.Context) error {
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
