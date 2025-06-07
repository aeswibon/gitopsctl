package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/git"
	"aeswibon.com/github/gitopsctl/internal/core/k8s"
	"go.uber.org/zap"
)

const (
	// MaxConsecutiveFailures defines the maximum number of consecutive failures
	// before the reconciliation loop stops for an application.
	MaxConsecutiveFailures = 5
	// BaseBackoffDuration defines the base duration for exponential backoff
	BaseBackoffDuration = 5 * time.Second
	// GitOperationTimeout defines the timeout for Git operations like clone/pull.
	GitOperationTimeout = 60 * time.Second
	// K8sApplyTimeout defines the timeout for applying Kubernetes manifests.
	K8sApplyTimeout = 120 * time.Second
	// K8sConnectTimeout defines the timeout for establishing a connection to the Kubernetes cluster.
	K8sConnectTimeout = 10 * time.Second
)

// Controller orchestrates the GitOps reconciliation loop.
//
// It manages the lifecycle of application synchronization processes.
type Controller struct {
	// Logger is used for structured logging throughout the controller.
	logger *zap.Logger
	// Apps holds the list of applications to be reconciled.
	apps *app.Applications
	// Context is used to manage cancellation and timeouts for the reconciliation loops.
	ctx context.Context
	// Cancel function to stop the context and signal all goroutines to exit.
	cancel context.CancelFunc
	// appContexts holds the context cancellation functions for each application.
	appContexts map[string]context.CancelFunc
	// mu protects the appContexts map to ensure thread-safe access.
	mu sync.Mutex
	// WaitGroup is used to wait for all reconciliation goroutines to finish before shutdown.
	wg sync.WaitGroup
}

// NewController creates a new Controller instance.
//
// It initializes the context and sets up the logger and applications.
func NewController(logger *zap.Logger, apps *app.Applications) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		logger:      logger,
		apps:        apps,
		ctx:         ctx,
		cancel:      cancel,
		appContexts: make(map[string]context.CancelFunc),
	}
}

// Start begins the reconciliation loop for all registered applications.
//
// It spawns a goroutine for each application to handle its synchronization process.
func (c *Controller) Start(appConfigFile string) error {
	c.logger.Info("Starting GitOps controller...")

	// Initial launch of goroutines for existing apps
	c.launchAppGoroutines(appConfigFile)

	c.logger.Info("All application reconciliation loops started.")
	return nil
}

// LaunchAppGoroutines starts goroutines for all applications currently in the app.Applications map.
//
// It checks if a goroutine is already running for each application to avoid duplicates.
func (c *Controller) launchAppGoroutines(appConfigFile string) {
	c.apps.RLock() // Acquire read lock to iterate apps
	appsToStart := c.apps.List()
	c.apps.RUnlock()

	for _, application := range appsToStart {
		c.mu.Lock() // Protect appContexts map
		_, exists := c.appContexts[application.Name]
		c.mu.Unlock()

		if exists {
			c.logger.Debug("Goroutine already running for application, skipping launch", zap.String("app", application.Name))
			continue // Skip if already running
		}

		// Create a separate context for each application's goroutine,
		// derived from the main controller context.
		appCtx, appCancel := context.WithCancel(c.ctx)
		c.mu.Lock()
		c.appContexts[application.Name] = appCancel // Store cancel function
		c.mu.Unlock()

		// Create a copy of the application for the goroutine to prevent data races
		appCopy := *application
		c.wg.Add(1)
		go c.reconcileApp(appCtx, &appCopy, appConfigFile, appCancel) // Pass appCancel for self-removal
	}
}

// Stop gracefully stops all reconciliation loops.
//
// It cancels the context and waits for all goroutines to finish.
func (c *Controller) Stop() {
	c.logger.Info("Stopping GitOps controller...")
	c.cancel()  // Signal all goroutines to stop
	c.wg.Wait() // Wait for all goroutines to finish
	c.logger.Info("GitOps controller stopped.")
}

// StopApp gracefully stops the reconciliation loop for a single application.
// This is called when an application is unregistered.
func (c *Controller) StopApp(appName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cancelFunc, ok := c.appContexts[appName]; ok {
		c.logger.Info("Stopping reconciliation loop for application", zap.String("app", appName))
		cancelFunc()                   // Signal the specific goroutine to stop
		delete(c.appContexts, appName) // Remove from map
	} else {
		c.logger.Warn("Attempted to stop non-existent or already stopped application goroutine", zap.String("app", appName))
	}
}

// ReconcileApp runs the GitOps loop for a single application.
//
// It handles Git repository synchronization and Kubernetes manifest application.
func (c *Controller) reconcileApp(appCtx context.Context, app *app.Application, appConfigFile string, appCancel context.CancelFunc) {
	defer c.wg.Done() // Decrement WaitGroup counter when the goroutine finishes
	// Ensure the app's cancel func is removed from the map when this goroutine exits
	defer func() {
		c.mu.Lock()
		delete(c.appContexts, app.Name)
		c.mu.Unlock()
		appCancel() // Also call the cancel func to ensure its context is marked done
	}()

	logger := c.logger.With(zap.String("app", app.Name))
	logger.Info("Starting reconciliation loop for application",
		zap.String("repo", app.RepoURL),
		zap.String("branch", app.Branch),
		zap.String("path", app.Path),
		zap.Duration("interval", app.PollingInterval))

	// Create a temporary directory for this app's Git repository
	repoDir, err := git.CreateTempRepoDir()
	if err != nil {
		logger.Error("Failed to create temporary repo directory", zap.Error(err))
		app.Status = "Error"
		app.Message = fmt.Sprintf("Failed to create temp dir: %v", err)
		c.saveAppStatus(app, appConfigFile, true) // Force save on critical error
		return
	}
	defer func() {
		// Clean up the temporary directory after use
		if cleanupErr := git.CleanUpRepo(logger, repoDir); cleanupErr != nil {
			logger.Error("Failed to clean up repo directory", zap.String("dir", repoDir), zap.Error(cleanupErr))
		}
	}()

	// Initialize Kubernetes client for this application
	k8sClient, err := k8s.NewClientSet(logger, app.KubeconfigPath)
	if err != nil {
		logger.Error("Failed to create Kubernetes client for application", zap.Error(err))
		app.Status = "Error"
		app.Message = fmt.Sprintf("Failed to create K8s client: %v", err)
		c.saveAppStatus(app, appConfigFile, true) // Force save on critical error
		return
	}

	// Perform an initial connectivity check with the Kubernetes cluster with a timeout
	// This ensures the controller can connect to the cluster before starting the reconciliation loop.
	// If the connection fails, we log the error and update the application's status accordingly.
	logger.Info("Checking connectivity to Kubernetes cluster", zap.String("kubeconfig", app.KubeconfigPath))
	connectCtx, connectCancel := context.WithTimeout(appCtx, K8sConnectTimeout)
	defer connectCancel()
	if err := k8sClient.CheckConnectivity(connectCtx); err != nil {
		logger.Error("Failed to connect to Kubernetes cluster", zap.Error(err))
		app.Status = "Error"
		app.Message = fmt.Sprintf("K8s connectivity error: %v", err)
		c.saveAppStatus(app, appConfigFile, true) // Force save on critical error
		return
	}

	// Initial sync attempt immediately
	c.performSync(appCtx, logger, app, repoDir, k8sClient, appConfigFile)

	// Set up a ticker for periodic polling of the Git repository
	ticker := time.NewTicker(app.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Calculate effective polling interval with backoff
			currentInterval := app.PollingInterval
			if app.ConsecutiveFailures > 0 {
				backoffFactor := time.Duration(1 << (app.ConsecutiveFailures - 1)) // Exponential backoff
				backoffDuration := min(BaseBackoffDuration*backoffFactor, currentInterval*MaxConsecutiveFailures)
				currentInterval = backoffDuration
				logger.Warn("Applying backoff due to previous failures",
					zap.Int("failures", app.ConsecutiveFailures),
					zap.Duration("nextInterval", currentInterval))
			}

			// Reset ticker with potentially new interval
			ticker.Reset(currentInterval)

			c.performSync(appCtx, logger, app, repoDir, k8sClient, appConfigFile)

		case <-appCtx.Done():
			logger.Info("Reconciliation loop stopping for application.", zap.String("reason", appCtx.Err().Error()))
			// Only update status if it's not already stopped or explicitly error
			if app.Status != "Stopped" && app.Status != "Error" {
				app.Status = "Stopped"
				app.Message = fmt.Sprintf("Controller shut down: %v", appCtx.Err())

				c.saveAppStatus(app, appConfigFile, true) // Force save on shutdown
			}
			return
		}
	}
}

// PerformSync checks the Git repository for changes and applies Kubernetes manifests.
//
// It updates the application's status and handles errors appropriately.
func (c *Controller) performSync(ctx context.Context, logger *zap.Logger, app *app.Application, repoDir string, k8sClient *k8s.ClientSet, appConfigFile string) {
	previousStatus := app.Status
	previousHash := app.LastSyncedGitHash

	logger.Debug("Polling Git repository...")
	currentHash, err := git.CloneOrPull(ctx, logger, app.RepoURL, app.Branch, repoDir)
	if err != nil {
		logger.Error("Failed to pull Git repository", zap.Error(err))
		app.Status = "Error"
		app.Message = fmt.Sprintf("Git pull error: %v", err)
		app.ConsecutiveFailures++
		c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash)
		return
	}

	if currentHash == app.LastSyncedGitHash {
		logger.Debug("No new changes detected in Git repository", zap.String("hash", currentHash))
		// Only change status to Synced if it was previously an error, otherwise keep it as is
		if app.Status == "Error" {
			app.Status = "Synced"
			app.Message = fmt.Sprintf("Up to date at %s", currentHash)
			app.ConsecutiveFailures = 0 // Reset failures on successful "check"
			c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash)
		} else {
			// No actual change, just update timestamp/message if desired, but don't force save
			// unless explicitly status changed.
			app.Message = fmt.Sprintf("Up to date at %s", currentHash)
		}
		return
	}

	logger.Info("New changes detected in Git repository",
		zap.String("oldHash", app.LastSyncedGitHash),
		zap.String("newHash", currentHash))

	manifestsDir := filepath.Join(repoDir, app.Path)
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		logger.Error("Manifests path does not exist in repository", zap.String("path", app.Path))
		app.Status = "Error"
		app.Message = fmt.Sprintf("Manifests path '%s' not found in repo after cloning. Check 'path' in config or repo structure.", app.Path)
		app.ConsecutiveFailures++
		c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash)
		return
	}

	logger.Info("Applying Kubernetes manifests...", zap.String("sourceDir", manifestsDir))
	k8sApplyCtx, k8sApplyCancel := context.WithTimeout(ctx, K8sApplyTimeout)
	defer k8sApplyCancel() // Ensure the context is cancelled after applying manifests
	applyErrors := k8sClient.ApplyManifests(k8sApplyCtx, manifestsDir)
	if len(applyErrors) > 0 {
		errorMessages := make([]string, len(applyErrors))
		for i, e := range applyErrors {
			errorMessages[i] = e.Error()
		}
		errMsg := fmt.Sprintf("Failed to apply %d manifest(s): %s", len(applyErrors), strings.Join(errorMessages, "; "))
		logger.Error("Failed to apply Kubernetes manifests", zap.String("details", errMsg))
		app.Status = "Error"
		app.Message = errMsg
		app.ConsecutiveFailures++
		c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash)
		return
	}

	app.LastSyncedGitHash = currentHash
	app.Status = "Synced"
	app.Message = fmt.Sprintf("Successfully synced to %s", currentHash)
	app.ConsecutiveFailures = 0 // Reset failures on successful sync
	logger.Info("Successfully applied Kubernetes manifests", zap.String("hash", currentHash))

	c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash)
}

// saveAppStatus is a helper to update and persist the application's status.
//
// It locks the applications list to ensure thread-safe updates.
func (c *Controller) saveAppStatus(appToSave *app.Application, appConfigFile string, forceSave bool) {
	c.apps.Lock()
	defer c.apps.Unlock()

	// Retrieve the original application from the map to compare and update
	originalApp, ok := c.apps.Apps[appToSave.Name]
	if !ok {
		c.logger.Error("Attempted to save status for unknown application", zap.String("app", appToSave.Name))
		return
	}

	// Check if actual status or hash changed, or if forced to save
	if forceSave ||
		originalApp.Status != appToSave.Status ||
		originalApp.LastSyncedGitHash != appToSave.LastSyncedGitHash ||
		originalApp.ConsecutiveFailures != appToSave.ConsecutiveFailures { // NEW: also save if failures change

		// Update the shared map with the current state of the goroutine's app copy
		originalApp.Status = appToSave.Status
		originalApp.Message = appToSave.Message
		originalApp.LastSyncedGitHash = appToSave.LastSyncedGitHash
		originalApp.ConsecutiveFailures = appToSave.ConsecutiveFailures // NEW: update failures

		if err := app.SaveApplications(c.apps, appConfigFile); err != nil {
			c.logger.Error("Failed to save application status to file", zap.Error(err))
		} else {
			c.logger.Debug("Application status saved to file", zap.String("app", appToSave.Name), zap.String("status", appToSave.Status))
		}
	} else {
		c.logger.Debug("No significant change to application status, skipping save", zap.String("app", appToSave.Name))
	}
}
