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
	// Base backoff duration for retrying failed operations.
	BaseBackoffDuration = 5 * time.Second
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
	// WaitGroup is used to wait for all reconciliation goroutines to finish before shutdown.
	wg sync.WaitGroup
}

// NewController creates a new Controller instance.
//
// It initializes the context and sets up the logger and applications.
func NewController(logger *zap.Logger, apps *app.Applications) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		logger: logger,
		apps:   apps,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the reconciliation loop for all registered applications.
//
// It spawns a goroutine for each application to handle its synchronization process.
func (c *Controller) Start(appConfigFile string) error {
	c.logger.Info("Starting GitOps controller...")

	// Iterate over all applications and start their reconciliation loops
	c.apps.RLock()
	appsToReconcile := c.apps.List()
	defer c.apps.RUnlock()
	for _, application := range appsToReconcile {
		// Create a copy of the application for the goroutine to prevent data races
		appCopy := *application
		c.wg.Add(1)                                // Increment the WaitGroup counter for each application
		go c.reconcileApp(&appCopy, appConfigFile) // Start the reconciliation loop in a new goroutine
	}

	c.logger.Info("All application reconciliation loops started.")
	return nil
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

// ReconcileApp runs the GitOps loop for a single application.
//
// It handles Git repository synchronization and Kubernetes manifest application.
func (c *Controller) reconcileApp(application *app.Application, appConfigFile string) {
	defer c.wg.Done() // Decrement WaitGroup counter when the goroutine finishes

	logger := c.logger.With(zap.String("app", application.Name))
	logger.Info("Starting reconciliation loop for application",
		zap.String("repo", application.RepoURL),
		zap.String("branch", application.Branch),
		zap.String("path", application.Path),
		zap.Duration("interval", application.PollingInterval))

	// Create a temporary directory for this app's Git repository
	repoDir, err := git.CreateTempRepoDir()
	if err != nil {
		logger.Error("Failed to create temporary repo directory", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("Failed to create temp dir: %v", err)
		c.saveAppStatus(application, appConfigFile, true) // Force save on critical error
		return
	}
	defer func() {
		// Clean up the temporary directory after use
		if cleanupErr := git.CleanUpRepo(logger, repoDir); cleanupErr != nil {
			logger.Error("Failed to clean up repo directory", zap.String("dir", repoDir), zap.Error(cleanupErr))
		}
	}()

	// Initialize Kubernetes client for this application
	k8sClient, err := k8s.NewClientSet(logger, application.KubeconfigPath)
	if err != nil {
		logger.Error("Failed to create Kubernetes client for application", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("Failed to create K8s client: %v", err)
		c.saveAppStatus(application, appConfigFile, true) // Force save on critical error
		return
	}

	// Perform an initial connectivity check with the Kubernetes cluster
	if err := k8sClient.CheckConnectivity(c.ctx); err != nil {
		logger.Error("Failed to connect to Kubernetes cluster", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("K8s connectivity error: %v", err)
		c.saveAppStatus(application, appConfigFile, true) // Force save on critical error
		return
	}

	// Initial sync attempt immediately
	c.performSync(logger, application, repoDir, k8sClient, appConfigFile)

	// Set up a ticker for periodic polling of the Git repository
	ticker := time.NewTicker(application.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Calculate effective polling interval with backoff
			currentInterval := application.PollingInterval
			if application.ConsecutiveFailures > 0 {
				backoffFactor := time.Duration(1 << (application.ConsecutiveFailures - 1)) // Exponential backoff
				backoffDuration := min(BaseBackoffDuration*backoffFactor, currentInterval*MaxConsecutiveFailures)
				currentInterval = backoffDuration
				logger.Warn("Applying backoff due to previous failures",
					zap.Int("failures", application.ConsecutiveFailures),
					zap.Duration("nextInterval", currentInterval))
			}

			// Reset ticker with potentially new interval
			ticker.Reset(currentInterval)

			c.performSync(logger, application, repoDir, k8sClient, appConfigFile)

		case <-c.ctx.Done():
			logger.Info("Reconciliation loop stopping for application.", zap.String("reason", c.ctx.Err().Error()))
			// Only update status if it's not already stopped or explicitly error
			if application.Status != "Stopped" && application.Status != "Error" {
				application.Status = "Stopped"
				application.Message = "Controller shut down."
				c.saveAppStatus(application, appConfigFile, true) // Force save on shutdown
			}
			return
		}
	}
}

// PerformSync checks the Git repository for changes and applies Kubernetes manifests.
//
// It updates the application's status and handles errors appropriately.
func (c *Controller) performSync(logger *zap.Logger, application *app.Application, repoDir string, k8sClient *k8s.ClientSet, appConfigFile string) {
	previousStatus := application.Status
	previousHash := application.LastSyncedGitHash

	logger.Debug("Polling Git repository...")
	currentHash, err := git.CloneOrPull(logger, application.RepoURL, application.Branch, repoDir) // NEW: Pass application.Branch
	if err != nil {
		logger.Error("Failed to pull Git repository", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("Git pull error: %v", err)
		application.ConsecutiveFailures++
		c.saveAppStatus(application, appConfigFile, previousStatus != application.Status || previousHash != application.LastSyncedGitHash)
		return
	}

	if currentHash == application.LastSyncedGitHash {
		logger.Debug("No new changes detected in Git repository", zap.String("hash", currentHash))
		// Only change status to Synced if it was previously an error, otherwise keep it as is
		if application.Status == "Error" {
			application.Status = "Synced"
			application.Message = fmt.Sprintf("Up to date at %s", currentHash)
			application.ConsecutiveFailures = 0 // Reset failures on successful "check"
			c.saveAppStatus(application, appConfigFile, previousStatus != application.Status || previousHash != application.LastSyncedGitHash)
		} else {
			// No actual change, just update timestamp/message if desired, but don't force save
			// unless explicitly status changed.
			application.Message = fmt.Sprintf("Up to date at %s", currentHash)
		}
		return
	}

	logger.Info("New changes detected in Git repository",
		zap.String("oldHash", application.LastSyncedGitHash),
		zap.String("newHash", currentHash))

	manifestsDir := filepath.Join(repoDir, application.Path)
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		logger.Error("Manifests path does not exist in repository", zap.String("path", application.Path))
		application.Status = "Error"
		application.Message = fmt.Sprintf("Manifests path '%s' not found in repo after cloning. Check 'path' in config or repo structure.", application.Path)
		application.ConsecutiveFailures++
		c.saveAppStatus(application, appConfigFile, previousStatus != application.Status || previousHash != application.LastSyncedGitHash)
		return
	}

	logger.Info("Applying Kubernetes manifests...", zap.String("sourceDir", manifestsDir))
	applyErrors := k8sClient.ApplyManifests(c.ctx, manifestsDir) // CHANGED: receives slice of errors
	if len(applyErrors) > 0 {
		errorMessages := make([]string, len(applyErrors))
		for i, e := range applyErrors {
			errorMessages[i] = e.Error()
		}
		errMsg := fmt.Sprintf("Failed to apply %d manifest(s): %s", len(applyErrors), strings.Join(errorMessages, "; "))
		logger.Error("Failed to apply Kubernetes manifests", zap.String("details", errMsg))
		application.Status = "Error"
		application.Message = errMsg
		application.ConsecutiveFailures++
		c.saveAppStatus(application, appConfigFile, previousStatus != application.Status || previousHash != application.LastSyncedGitHash)
		return
	}

	application.LastSyncedGitHash = currentHash
	application.Status = "Synced"
	application.Message = fmt.Sprintf("Successfully synced to %s", currentHash)
	application.ConsecutiveFailures = 0 // Reset failures on successful sync
	logger.Info("Successfully applied Kubernetes manifests", zap.String("hash", currentHash))

	c.saveAppStatus(application, appConfigFile, previousStatus != application.Status || previousHash != application.LastSyncedGitHash)
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
