package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultAppConfigFile is the default path to store registered applications
	DefaultAppConfigFile = "configs/applications.json"
)

// Application represents a single GitOps application managed by the controller.
//
// It encapsulates all the necessary metadata and operational details required
// to monitor and synchronize the application's state between Git and Kubernetes.
type Application struct {
	// Name is a unique identifier for the application.
	//
	// It must be unique across all registered applications and should follow
	// DNS subdomain naming conventions for compatibility with Kubernetes resources.
	Name string `json:"name"`

	// RepoURL specifies the URL of the Git repository where the application's manifests are stored.
	//
	// This URL can be HTTPS or SSH-based, depending on the user's authentication setup.
	RepoURL string `json:"repoURL"`

	// Branch defines the Git branch to monitor for changes.
	//
	// The controller will track this branch for updates and apply changes accordingly.
	Branch string `json:"branch"`

	// Path specifies the relative directory within the repository where Kubernetes manifests are located.
	//
	// This allows users to organize multiple applications or environments within a single repository.
	Path string `json:"path"`

	// ClusterName is the name of the Kubernetes cluster where the application will be deployed.
	//
	// This name is used for logging and status reporting purposes.
	ClusterName string `json:"clusterName"`

	// Interval is the polling interval as a string (e.g., "5m", "30s").
	//
	// It defines how frequently the controller should check the Git repository for changes.
	Interval string `json:"interval"`

	// PollingInterval is the parsed duration of the Interval field for internal use.
	//
	// This field is not serialized into JSON and is used for efficient time-based operations.
	PollingInterval time.Duration `json:"-"`

	// LastSyncedGitHash stores the Git commit hash of the last successfully synchronized state.
	//
	// This helps the controller detect changes and avoid redundant operations.
	LastSyncedGitHash string `json:"lastSyncedGitHash,omitempty"`

	// Status represents the current operational state of the application.
	//
	// Possible values include "Running", "Error", "Synced", "Pending", etc.
	Status string `json:"status,omitempty"`

	// Message provides additional context about the application's current state.
	//
	// It can include error details, success messages, or other relevant information.
	Message string `json:"message,omitempty"`

	// ConsecutiveFailures tracks the number of consecutive synchronization failures.
	//
	// This can be used to implement backoff logic or alerting mechanisms.
	ConsecutiveFailures int `json:"consecutiveFailures,omitempty"`
}

// Applications represents a collection of Application objects.
//
// It uses a mutex to ensure thread-safe access to the underlying map of applications.
type Applications struct {
	Apps map[string]*Application
	mu   sync.RWMutex
}

// NewApplications creates and initializes a new Applications collection.
//
// It returns an empty collection with a properly initialized map.
func NewApplications() *Applications {
	return &Applications{
		Apps: make(map[string]*Application),
	}
}

// Lock acquires a write lock on the Applications collection.
//
// This ensures exclusive access to the collection for write operations.
func (a *Applications) Lock() {
	a.mu.Lock()
}

// RLock acquires a read lock on the Applications collection.
//
// This allows multiple readers to access the collection concurrently,
// while preventing write operations until the read lock is released.
func (a *Applications) RLock() {
	a.mu.RLock()
}

// RUnlock releases the read lock held on the Applications collection.
//
// It should always be called after RLock, typically using a defer statement.
func (a *Applications) RUnlock() {
	a.mu.RUnlock()
}

// Unlock releases the write lock held on the Applications collection.
//
// It should always be called after Lock, typically using a defer statement.
func (a *Applications) Unlock() {
	a.mu.Unlock()
}

// Add adds a new application to the collection.
//
// The caller is responsible for acquiring the necessary write lock before calling this method.
func (a *Applications) Add(app *Application) {
	a.Apps[app.Name] = app
}

// Get retrieves an application by its name.
//
// The caller is responsible for acquiring the necessary read or write lock before calling this method.
func (a *Applications) Get(name string) (*Application, bool) {
	app, ok := a.Apps[name]
	return app, ok
}

// List returns a slice containing all applications in the collection.
//
// The caller is responsible for acquiring the necessary read or write lock before calling this method.
func (a *Applications) List() []*Application {
	list := make([]*Application, 0, len(a.Apps))
	for _, app := range a.Apps {
		list = append(list, app)
	}
	return list
}

// Delete removes an application from the collection by its name.
//
// The caller is responsible for acquiring the necessary write lock before calling this method.
func (a *Applications) Delete(name string) {
	delete(a.Apps, name)
}

// LoadApplications loads applications from the specified JSON file.
// It initializes the Applications collection and populates it with data from the file.
// If the file does not exist, it returns an empty collection.
func LoadApplications(filePath string) (*Applications, error) {
	apps := NewApplications()
	apps.mu.Lock() // Acquire lock for initial load
	defer apps.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return apps, nil // Return empty if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read applications file %s: %w", filePath, err)
	}

	var loadedApps []*Application
	if err := json.Unmarshal(data, &loadedApps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal applications data: %w", err)
	}

	for _, app := range loadedApps {
		// Parse interval string to time.Duration
		duration, err := time.ParseDuration(app.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid polling interval for app %s: %w", app.Name, err)
		}
		app.PollingInterval = duration
		apps.Apps[app.Name] = app // Directly add to map while lock is held
	}

	return apps, nil
}

// SaveApplications saves the current state of applications to the specified JSON file.
//
// The caller is responsible for acquiring the necessary lock before calling this method.
func SaveApplications(apps *Applications, filePath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Convert map to slice for stable JSON output
	list := make([]*Application, 0, len(apps.Apps))
	for _, app := range apps.Apps {
		list = append(list, app)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal applications data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write applications file %s: %w", filePath, err)
	}
	return nil
}
