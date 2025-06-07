package k8s

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
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
)

// ClientSet holds Kubernetes clients for dynamic interactions.
//
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
//
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
//
// This function processes all YAML files in the specified directory, decodes them into
// Kubernetes objects, and applies them to the cluster. It handles both creation and updates
// of resources based on their existence in the cluster.
func (cs *ClientSet) ApplyManifests(ctx context.Context, manifestsDir string) []error {
	cs.logger.Info("Applying manifests", zap.String("directory", manifestsDir))
	var applyErrors []error

	err := filepath.WalkDir(manifestsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // Skip directories
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil // Only process YAML files
		}

		cs.logger.Debug("Processing manifest file", zap.String("file", path))
		data, err := os.ReadFile(path)
		if err != nil {
			cs.logger.Error("Failed to read manifest file", zap.String("file", path), zap.Error(err))
			applyErrors = append(applyErrors, fmt.Errorf("failed to read file %s: %w", path, err))
			return nil // Continue processing other files even if one fails
		}

		// Split multi-document YAML files
		decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		objects := strings.Split(string(data), "\n---")

		for _, objStr := range objects {
			if strings.TrimSpace(objStr) == "" {
				continue
			}

			unstructuredObj := &unstructured.Unstructured{}
			_, gvk, err := decoder.Decode([]byte(objStr), nil, unstructuredObj)
			if err != nil {
				cs.logger.Error("Failed to decode YAML object", zap.String("file", path), zap.Error(err))
				applyErrors = append(applyErrors, fmt.Errorf("failed to decode YAML object in file %s: %w", path, err))
				continue // Skip this object and continue with the next
			}

			mapping, err := cs.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				cs.logger.Error("Failed to get REST mapping for GVK",
					zap.String("gvk", gvk.String()), zap.Error(err))
				applyErrors = append(applyErrors, fmt.Errorf("failed to get REST mapping for GVK %s: %w", gvk.String(), err))
				continue // Skip this object and continue with the next
			}

			var dr dynamic.ResourceInterface
			if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
				// namespaced resources should specify the namespace
				if unstructuredObj.GetNamespace() == "" {
					unstructuredObj.SetNamespace("default") // Default to 'default' if not specified
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
						zap.Error(createErr))
					applyErrors = append(applyErrors, fmt.Errorf("failed to create %s %s: %w", gvk.Kind, unstructuredObj.GetName(), createErr))
					continue
				}
				cs.logger.Info("Created resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()))
			} else {
				// Resource exists, update it (using server-side apply equivalent)
				// For simplicity in MVP, we'll do a simple update.
				// Proper server-side apply would use unstructured.Unstructured.Object and field managers.
				unstructuredObj.SetResourceVersion("") // Clear resource version for update
				_, updateErr := dr.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
				if updateErr != nil {
					cs.logger.Error("Failed to update resource",
						zap.String("kind", gvk.Kind),
						zap.String("name", unstructuredObj.GetName()),
						zap.Error(updateErr))
					applyErrors = append(applyErrors, fmt.Errorf("failed to update %s %s: %w", gvk.Kind, unstructuredObj.GetName(), updateErr))
					continue
				}
				cs.logger.Info("Updated resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()))
			}
		}
		return nil
	})
	if err != nil {
		cs.logger.Error("Failed to walk through manifests directory", zap.String("directory", manifestsDir), zap.Error(err))
		applyErrors = append(applyErrors, fmt.Errorf("failed to walk through manifests directory %s: %w", manifestsDir, err))
	}
	return applyErrors
}

// CheckConnectivity verifies connectivity to the Kubernetes cluster.
//
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
