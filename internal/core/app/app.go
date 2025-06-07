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
type Application struct {
	Name              string        `json:"name"`
	RepoURL           string        `json:"repoURL"`
	Path              string        `json:"path"`                        // Path within the repo where manifests are (e.g., "k8s/prod")
	KubeconfigPath    string        `json:"kubeconfigPath"`              // Path to the kubeconfig file
	Interval          string        `json:"interval"`                    // Polling interval (e.g., "5m", "30s")
	PollingInterval   time.Duration `json:"-"`                           // Parsed duration for internal use
	LastSyncedGitHash string        `json:"lastSyncedGitHash,omitempty"` // For tracking
	Status            string        `json:"status,omitempty"`            // "Running", "Error", "Stopped", "Synced", "Pending"
	Message           string        `json:"message,omitempty"`           // Last error or success message
}

// Applications represents a collection of Application objects, protected by a mutex.
// The mutex protects access to the 'Apps' map itself.
type Applications struct {
	Apps map[string]*Application
	mu   sync.RWMutex
}

// NewApplications creates a new empty Applications collection.
func NewApplications() *Applications {
	return &Applications{
		Apps: make(map[string]*Application),
	}
}

// Lock acquires a write lock on the Applications collection.
func (a *Applications) Lock() {
	a.mu.Lock()
}

// Unlock releases the write lock on the Applications collection.
func (a *Applications) Unlock() {
	a.mu.Unlock()
}

// Add adds an application to the collection.
// Caller is responsible for locking the Applications struct.
func (a *Applications) Add(app *Application) {
	// Assumes the caller has already acquired a write lock (a.mu.Lock())
	// or that this function is called during initialization before concurrency starts.
	a.Apps[app.Name] = app
}

// Get retrieves an application by name.
// Caller is responsible for locking the Applications struct.
func (a *Applications) Get(name string) (*Application, bool) {
	// Assumes the caller has already acquired a read or write lock (a.mu.RLock() or a.mu.Lock()).
	app, ok := a.Apps[name]
	return app, ok
}

// List returns a slice of all applications.
// Caller is responsible for locking the Applications struct.
func (a *Applications) List() []*Application {
	// Assumes the caller has already acquired a read or write lock (a.mu.RLock() or a.mu.Lock()).
	list := make([]*Application, 0, len(a.Apps))
	for _, app := range a.Apps {
		list = append(list, app)
	}
	return list
}

// Delete removes an application by name.
// Caller is responsible for locking the Applications struct.
func (a *Applications) Delete(name string) {
	// Assumes the caller has already acquired a write lock (a.mu.Lock()).
	delete(a.Apps, name)
}

// LoadApplications loads applications from the specified file path.
// This function acquires its own lock as it's typically called at startup.
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

// SaveApplications saves the current state of applications to the specified file path.
// This function assumes the caller has already acquired the necessary lock (e.g., a.mu.Lock()).
func SaveApplications(apps *Applications, filePath string) error {
	// IMPORTANT: No locking here. The caller (e.g., controller.saveAppStatus)
	// is responsible for acquiring the appropriate lock on the 'apps' struct.

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
