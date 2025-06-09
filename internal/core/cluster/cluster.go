package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultClusterConfigFile is the default path to store registered clusters
	DefaultClusterConfigFile = "configs/clusters.json"
)

// Cluster represents a registered Kubernetes cluster.
//
// It contains the cluster name, path to the kubeconfig file, registration time,
// and optional status and message fields for error handling or status reporting.
type Cluster struct {
	// Name is the unique identifier for the cluster.
	Name string `json:"name"`
	// KubeconfigPath is the path to the kubeconfig file for this cluster.
	KubeconfigPath string `json:"kubeconfigPath"`
	// RegisteredAt is the time when the cluster was registered.
	RegisteredAt time.Time `json:"registeredAt"`
	// Status and Message are optional fields for reporting the cluster's status.
	Status string `json:"status,omitempty"`
	// Message can contain additional information about the cluster's status.
	Message string `json:"message,omitempty"`
}

// Clusters represents a thread-safe collection of Cluster objects.
//
// It provides methods to add, retrieve, list, and delete clusters.
// The collection is protected by a read-write mutex to allow concurrent access.
type Clusters struct {
	Cs map[string]*Cluster
	mu sync.RWMutex
}

// NewClusters creates a new empty Clusters collection.
//
// It initializes the map to hold Cluster objects and returns a pointer to the Clusters instance.
func NewClusters() *Clusters {
	return &Clusters{
		Cs: make(map[string]*Cluster),
	}
}

// Lock method acquires a write lock on the Clusters collection.
//
// This prevents other goroutines from reading or writing to the collection
// while the lock is held, ensuring thread safety during modifications.
func (c *Clusters) Lock() {
	c.mu.Lock()
}

// RLock method acquires a read lock on the Clusters collection.
//
// This allows multiple readers to access the collection concurrently,
// but prevents any writes while the read lock is held.
func (c *Clusters) RLock() {
	c.mu.RLock()
}

// Unlock method releases the write lock on the Clusters collection.
//
// It should be called after any modifications to the collection.
func (c *Clusters) Unlock() {
	c.mu.Unlock()
}

// RUnlock method releases the read lock on the Clusters collection.
//
// It should be called after read operations on the collection.
func (c *Clusters) RUnlock() {
	c.mu.RUnlock()
}

// Add adds a cluster to the collection.
//
// If a cluster with the same name already exists, it will be overwritten.
// This method is not thread-safe and should be called with the write lock held.
func (c *Clusters) Add(cluster *Cluster) {
	c.Cs[cluster.Name] = cluster
}

// Get retrieves a cluster by name.
//
// If the cluster exists, it returns the cluster and true.
// If the cluster does not exist, it returns nil and false.
func (c *Clusters) Get(name string) (*Cluster, bool) {
	cluster, ok := c.Cs[name]
	return cluster, ok
}

// List returns a slice of all clusters.
//
// It returns a slice of pointers to Cluster objects.
func (c *Clusters) List() []*Cluster {
	list := make([]*Cluster, 0, len(c.Cs))
	for _, cluster := range c.Cs {
		list = append(list, cluster)
	}
	return list
}

// Delete removes a cluster by name.
//
// If the cluster does not exist, it does nothing.
func (c *Clusters) Delete(name string) {
	delete(c.Cs, name)
}

// LoadClusters loads clusters from the specified file path.
//
// It reads the JSON data from the file, unmarshals it into Cluster objects,
// and populates the Clusters collection. If the file does not exist, it returns an empty collection.
// This function acquires its own lock as it's typically called at startup.
func LoadClusters(filePath string) (*Clusters, error) {
	clusters := NewClusters()
	clusters.mu.Lock() // Acquire lock for initial load
	defer clusters.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return clusters, nil // Return empty if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read clusters file %s: %w", filePath, err)
	}

	var loadedClusters []*Cluster
	if err := json.Unmarshal(data, &loadedClusters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clusters data: %w", err)
	}

	for _, cluster := range loadedClusters {
		clusters.Cs[cluster.Name] = cluster // Directly add to map while lock is held
	}

	return clusters, nil
}

// SaveClusters saves the current state of clusters to the specified file path.
//
// It serializes the Clusters collection to JSON and writes it to the file.
// If the directory does not exist, it creates it.
// This function does not acquire its own lock, so it should be called with the appropriate lock held.
func SaveClusters(clusters *Clusters, filePath string) error {
	// IMPORTANT: No locking here. The caller is responsible for acquiring the appropriate lock.

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	list := make([]*Cluster, 0, len(clusters.Cs))
	for _, cluster := range clusters.Cs {
		list = append(list, cluster)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal clusters data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write clusters file %s: %w", filePath, err)
	}
	return nil
}
