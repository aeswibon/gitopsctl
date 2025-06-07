package k8s

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

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

const (
	// DefaultAPITimeout is the default timeout for Kubernetes API requests
	DefaultAPITimeout = 30 * time.Second
	// DefaultQPS is the default queries per second for client-go
	DefaultQPS = 100
	// DefaultBurst is the default burst for client-go
	DefaultBurst = 100
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
//
// This function processes all YAML files in the specified directory, decodes them into
// Kubernetes objects, and applies them to the cluster. It handles both creation and updates
// of resources based on their existence in the cluster.
func (cs *ClientSet) ApplyManifests(ctx context.Context, manifestsDir string) []error {
	cs.logger.Info("Applying manifests", zap.String("directory", manifestsDir))
	var applyErrors []error

	err := filepath.WalkDir(manifestsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			applyErrors = append(applyErrors, fmt.Errorf("filesystem error walking %s: %w", path, err))
			return nil // Continue walking other files but log the error
		}
		if d.IsDir() {
			return nil // Skip directories
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil // Only process YAML files
		}

		cs.logger.Debug("Processing manifest file", zap.String("file", path))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			cs.logger.Error("Failed to read manifest file", zap.String("file", path), zap.Error(readErr))
			applyErrors = append(applyErrors, fmt.Errorf("failed to read file %s: %w", path, readErr))
			return nil // Continue to next file
		}

		// Split multi-document YAML files
		decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		objects := strings.Split(string(data), "\n---")

		for i, objStr := range objects {
			// Skip empty documents
			trimmedObjStr := strings.TrimSpace(objStr)
			if trimmedObjStr == "" {
				continue
			}

			unstructuredObj := &unstructured.Unstructured{}
			_, gvk, decodeErr := decoder.Decode([]byte(trimmedObjStr), nil, unstructuredObj)
			if decodeErr != nil {
				cs.logger.Error("Failed to decode YAML object", zap.String("file", path), zap.Int("documentIdx", i), zap.Error(decodeErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to decode YAML from %s (doc %d): %w", path, i, decodeErr))
				continue // Continue to next document
			}

			if unstructuredObj.GetName() == "" {
				cs.logger.Warn("Skipping unnamed resource in manifest", zap.String("file", path), zap.Int("documentIdx", i), zap.String("kind", gvk.Kind))
				applyErrors = append(applyErrors, fmt.Errorf("skipping unnamed resource in %s (doc %d) of kind %s", path, i, gvk.Kind))
				continue
			}

			mapping, mappingErr := cs.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if mappingErr != nil {
				cs.logger.Error("Failed to get REST mapping for GVK",
					zap.String("gvk", gvk.String()), zap.String("file", path), zap.Error(mappingErr))
				applyErrors = append(applyErrors, fmt.Errorf("failed to get REST mapping for %s in %s: %w", gvk.String(), path, mappingErr))
				continue // Continue to next document
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
						zap.String("namespace", unstructuredObj.GetNamespace()),
						zap.Error(createErr))
					applyErrors = append(applyErrors, fmt.Errorf("failed to create %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), path, createErr))
					continue // Continue to next document
				}
				cs.logger.Info("Created resource",
					zap.String("kind", gvk.Kind),
					zap.String("name", unstructuredObj.GetName()),
					zap.String("namespace", unstructuredObj.GetNamespace()))
			} else {
				// Resource exists, update it (using simple update for MVP)
				// For proper server-side apply, you'd use FieldManager and Apply method
				// unstructuredObj.SetResourceVersion("") // Clear resource version for update (optional, usually handled by server-side apply)
				_, updateErr := dr.Update(ctx, unstructuredObj, metav1.UpdateOptions{})
				if updateErr != nil {
					cs.logger.Error("Failed to update resource",
						zap.String("kind", gvk.Kind),
						zap.String("name", unstructuredObj.GetName()),
						zap.String("namespace", unstructuredObj.GetNamespace()),
						zap.Error(updateErr))
					applyErrors = append(applyErrors, fmt.Errorf("failed to update %s %s/%s from %s: %w", gvk.Kind, unstructuredObj.GetNamespace(), unstructuredObj.GetName(), path, updateErr))
					continue // Continue to next document
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
		applyErrors = append(applyErrors, fmt.Errorf("error during manifest directory walk %s: %w", manifestsDir, err))
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
