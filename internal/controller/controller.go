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
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"aeswibon.com/github/gitopsctl/internal/core/git"
	"aeswibon.com/github/gitopsctl/internal/core/k8s"
	"go.uber.org/zap"
)

// ClusterCommandType defines the type of command for a cluster.
type ClusterCommandType string

// ClusterCommand represents a command to be executed for a specific cluster.
type ClusterCommand struct {
	Type        ClusterCommandType
	ClusterName string
}

const (
	// ClusterCommandCheck indicates a command to check the health of a cluster.
	// This is used to trigger a health check for the cluster's connectivity and status.
	ClusterCommandCheck ClusterCommandType = "CHECK"
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

// AppCommandType defines the type of command for an application.
type AppCommandType string

const (
	// AppCommandStart indicates a command to start or restart an app's reconciliation.
	// This is used to trigger the reconciliation loop for an application.
	AppCommandStart AppCommandType = "START"
	// AppCommandStop indicates a command to stop an app's reconciliation.
	// This is used to gracefully stop the reconciliation loop for an application.
	AppCommandStop AppCommandType = "STOP"
	// AppCommandSync indicates a command to trigger an immediate sync for an app.
	// This is used to force a synchronization of the application's Git repository
	AppCommandSync AppCommandType = "SYNC"
)

// AppCommand represents a command to be executed for a specific application.
//
// It includes the type of command (start, stop, sync) and the application name.
// The Data field can be used for additional parameters if needed, such as force sync or specific commit.
type AppCommand struct {
	// Type of command to execute for the application.
	Type AppCommandType
	// AppName is the name of the application to which this command applies.
	AppName string
	// Data can be used for additional parameters if needed (e.g., force sync, specific commit)
	Data map[string]any
}

// AppRuntime holds the context and cancel function for a running application goroutine.
//
// It is used to manage the lifecycle of the application's reconciliation loop.
type appRuntime struct {
	// Context is used to manage the lifecycle of the application's reconciliation loop.
	cancel context.CancelFunc
	// syncChan is a channel used to trigger immediate synchronization of the application.
	syncChan chan struct{}
}

// Controller orchestrates the GitOps reconciliation loop.
//
// It manages the lifecycle of application synchronization processes.
type Controller struct {
	// Logger is used for structured logging throughout the controller.
	logger *zap.Logger
	// Apps holds the list of applications to be reconciled.
	apps *app.Applications
	// Clusters holds the list of clusters to which applications can be deployed.
	clusters *cluster.Clusters
	// Context is used to manage cancellation and timeouts for the reconciliation loops.
	ctx context.Context
	// Cancel function to stop the context and signal all goroutines to exit.
	cancel context.CancelFunc
	// AppCommandChan is a channel for receiving commands to start, stop, or sync applications.
	appCommandChan chan AppCommand
	// ClusterCommandChan is a channel for receiving commands related to cluster health checks.
	clusterCommandChan chan ClusterCommand
	// RunningApps holds the currently running applications and their contexts.
	runningApps map[string]*appRuntime
	// mu protects the appContexts map to ensure thread-safe access.
	mu sync.Mutex
	// WaitGroup is used to wait for all reconciliation goroutines to finish before shutdown.
	wg sync.WaitGroup
}

// NewController creates a new Controller instance.
//
// It initializes the context and sets up the logger and applications.
func NewController(logger *zap.Logger, apps *app.Applications, clusters *cluster.Clusters) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		logger:             logger,
		apps:               apps,
		clusters:           clusters,
		ctx:                ctx,
		cancel:             cancel,
		appCommandChan:     make(chan AppCommand, 10),
		clusterCommandChan: make(chan ClusterCommand, 10),
		runningApps:        make(map[string]*appRuntime),
	}
}

// Start begins the reconciliation loop for all registered applications.
//
// It spawns a goroutine for each application to handle its synchronization process.
func (c *Controller) Start(appConfigFile string) error {
	c.logger.Info("Starting GitOps controller...")

	c.wg.Add(1)
	go c.commandDispatcher(appConfigFile)

	c.wg.Add(1)
	go c.clusterHealthChecker()

	c.apps.RLock()
	defer c.apps.RUnlock()

	appsToStart := c.apps.List()
	if len(appsToStart) > 0 {
		c.logger.Info(fmt.Sprintf("Attempting to launch %d existing application reconciliation loops...", len(appsToStart)))
		for _, application := range appsToStart {
			c.appCommandChan <- AppCommand{Type: AppCommandStart, AppName: application.Name}
		}
	} else {
		c.logger.Info("No existing applications found to launch at startup.")
	}

	c.clusters.RLock()
	defer c.clusters.RUnlock()

	clustersToCheck := c.clusters.List()
	if len(clustersToCheck) > 0 {
		c.logger.Info(fmt.Sprintf("Triggering initial health checks for %d clusters...", len(clustersToCheck)))
		for _, cl := range clustersToCheck {
			c.clusterCommandChan <- ClusterCommand{Type: ClusterCommandCheck, ClusterName: cl.Name}
		}
	} else {
		c.logger.Info("No existing clusters found to check at startup.")
	}

	c.logger.Info("Initial application reconciliation loops dispatched.")
	return nil
}

// Stop gracefully stops all reconciliation loops.
//
// It cancels the context and waits for all goroutines to finish.
func (c *Controller) Stop() {
	c.logger.Info("Stopping GitOps controller...")
	c.cancel()                  // Signal all goroutines to stop
	close(c.appCommandChan)     // Close the command channel
	close(c.clusterCommandChan) // Close the cluster command channel
	c.wg.Wait()                 // Wait for all goroutines to finish
	c.logger.Info("GitOps controller stopped.")
}

// StartApp sends a command to start or restart an application's reconciliation loop.
//
// It will reload the application's definition from the config file.
func (c *Controller) StartApp(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandStart, AppName: appName}
}

// StopApp sends a command to stop an application's reconciliation loop.
//
// It will gracefully stop the reconciliation loop for the specified application.
func (c *Controller) StopApp(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandStop, AppName: appName}
}

// TriggerSync sends a command to trigger an immediate sync for an application.
//
// This is useful for forcing a synchronization of the application's Git repository
func (c *Controller) TriggerSync(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandSync, AppName: appName}
}

// TriggerClusterHealthCheck sends a command to trigger an immediate health check for a cluster.
//
// This is useful for manually checking the connectivity and status of a cluster.
func (c *Controller) TriggerClusterHealthCheck(clusterName string) {
	c.clusterCommandChan <- ClusterCommand{Type: ClusterCommandCheck, ClusterName: clusterName}
}

// CommandDispatcher is the central goroutine that processes application commands.
//
// It listens for commands to start, stop, or sync applications and manages their reconciliation loops.
func (c *Controller) commandDispatcher(appConfigFile string) {
	defer c.wg.Done()
	c.logger.Info("Starting controller command dispatcher...")

	for {
		select {
		case cmd, ok := <-c.appCommandChan:
			if !ok { // Channel closed, dispatcher should exit
				c.logger.Info("Command channel closed, dispatcher exiting.")
				c.stopAllAppGoroutines() // Stop any remaining app goroutines
				return
			}
			c.handleAppCommand(cmd, appConfigFile)
		case <-c.ctx.Done(): // Main controller context cancelled, dispatcher should exit
			c.logger.Info("Main controller context cancelled, dispatcher exiting.")
			c.stopAllAppGoroutines() // Stop any remaining app goroutines
			return
		}
	}
}

// ClusterHealthChecker periodically checks the health of registered clusters.
//
// It runs in a separate goroutine and checks cluster connectivity at regular intervals.
func (c *Controller) clusterHealthChecker() {
	defer c.wg.Done()
	c.logger.Info("Cluster health checker started.")

	ticker := time.NewTicker(cluster.DefaultClusterHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.clusters.RLock()
			defer c.clusters.RUnlock()

			clustersToCheck := c.clusters.List()
			for _, cl := range clustersToCheck {
				c.performClusterHealthCheck(c.ctx, cl)
			}
		case cmd, ok := <-c.clusterCommandChan:
			if !ok {
				c.logger.Info("Cluster command channel closed, health checker exiting.")
				return
			}
			if cmd.Type == ClusterCommandCheck {
				cl, exists := c.clusters.Get(cmd.ClusterName)
				if exists {
					c.logger.Info("Manual health check triggered for cluster", zap.String("cluster", cmd.ClusterName))
					c.performClusterHealthCheck(c.ctx, cl)
				} else {
					c.logger.Warn("Attempted manual health check for non-existent cluster", zap.String("cluster", cmd.ClusterName))
				}
			}
		case <-c.ctx.Done():
			c.logger.Info("Main controller context cancelled, cluster health checker exiting.")
			return
		}
	}
}

// PerformClusterHealthCheck performs a connectivity check for a given cluster and updates its status.
//
// It creates a Kubernetes client for the cluster and checks connectivity.
func (c *Controller) performClusterHealthCheck(ctx context.Context, cl *cluster.Cluster) {
	logger := c.logger.With(zap.String("cluster", cl.Name))
	logger.Debug("Performing health check for cluster.")

	// Create a client for the specific cluster
	k8sClient, err := k8s.NewClientSet(logger, cl.KubeconfigPath)
	if err != nil {
		logger.Error("Failed to create K8s client for cluster health check", zap.Error(err))
		cl.Status = "Error"
		cl.Message = fmt.Sprintf("Failed to create K8s client: %v", err)
	} else {
		checkCtx, checkCancel := context.WithTimeout(ctx, K8sConnectTimeout)
		defer checkCancel()
		if err := k8sClient.CheckConnectivity(checkCtx); err != nil {
			logger.Warn("Cluster connectivity check failed", zap.Error(err))
			cl.Status = "Unreachable"
			cl.Message = fmt.Sprintf("Connectivity failed: %v", err)
		} else {
			logger.Debug("Cluster connectivity check successful.")
			cl.Status = "Active"
			cl.Message = "Connectivity successful."
		}
	}
	cl.LastCheckedAt = time.Now()

	// Save cluster status
	c.clusters.Lock()
	if err := cluster.SaveClusters(c.clusters, cluster.DefaultClusterConfigFile); err != nil {
		logger.Error("Failed to save cluster status to file", zap.Error(err))
	}
	c.clusters.Unlock()
}

// HandleAppCommand processes a single application command.
//
// It starts, stops, or syncs the specified application based on the command type.
func (c *Controller) handleAppCommand(cmd AppCommand, appConfigFile string) {
	c.logger.Debug("Received app command", zap.String("type", string(cmd.Type)), zap.String("app", cmd.AppName))

	c.mu.Lock()
	defer c.mu.Unlock()

	switch cmd.Type {
	case AppCommandStart:
		// Load the application config fresh in case it was updated
		c.apps.RLock()
		defer c.apps.RUnlock()

		appConfig, exists := c.apps.Get(cmd.AppName)
		if !exists {
			c.logger.Error("Attempted to start non-existent application", zap.String("app", cmd.AppName))
			return
		}

		c.clusters.RLock()
		defer c.clusters.RUnlock()

		_, clusterExists := c.clusters.Get(appConfig.ClusterName)
		if !clusterExists {
			c.logger.Error("Attempted to start application with non-existent cluster",
				zap.String("app", cmd.AppName),
				zap.String("cluster", appConfig.ClusterName))

			appConfig.Status = "Error"
			appConfig.Message = fmt.Sprintf("Cluster '%s' does not exist", appConfig.ClusterName)
			appConfig.ConsecutiveFailures = 0               // Reset failures on critical error
			c.saveAppStatus(appConfig, appConfigFile, true) // Force save on critical error
			return
		}

		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			// If already running, stop the old one to restart with new config
			c.logger.Info("Restarting application reconciliation loop", zap.String("app", cmd.AppName))
			runtime.cancel() // Cancel the old context
			// The deferred func in reconcileApp will clean up the old entry from runningApps
		}

		appCtx, appCancel := context.WithCancel(c.ctx) // New context for the app
		syncChan := make(chan struct{}, 1)             // New sync channel for the app

		appCopy := *appConfig // Create a copy for the goroutine
		c.wg.Add(1)
		c.runningApps[cmd.AppName] = &appRuntime{cancel: appCancel, syncChan: syncChan}
		go c.reconcileApp(appCtx, &appCopy, appConfigFile, appCancel, syncChan)

	case AppCommandStop:
		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			c.logger.Info("Stopping application reconciliation loop", zap.String("app", cmd.AppName))
			runtime.cancel() // Cancel the specific app's context
			// The deferred func in reconcileApp will clean up the old entry from runningApps
		} else {
			c.logger.Warn("Attempted to stop non-running application", zap.String("app", cmd.AppName))
		}

	case AppCommandSync:
		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			select {
			case runtime.syncChan <- struct{}{}:
				c.logger.Info("Manual sync signal sent to application", zap.String("app", cmd.AppName))
			default:
				c.logger.Warn("Application sync channel is busy, skipping immediate sync", zap.String("app", cmd.AppName))
			}
		} else {
			c.logger.Warn("Attempted to trigger sync for non-running application", zap.String("app", cmd.AppName))
		}
	}
}

// StopAllAppGoroutines iterates and stops all currently running application goroutines.
//
// It cancels their contexts and removes them from the runningApps map.
func (c *Controller) stopAllAppGoroutines() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for appName, runtime := range c.runningApps {
		c.logger.Info("Stopping all application reconciliation loops during shutdown", zap.String("app", appName))
		runtime.cancel()
		// The deferred func in reconcileApp will clean up the old entry from runningApps
	}
}

// ReconcileApp runs the GitOps loop for a single application.
//
// It handles Git repository synchronization and Kubernetes manifest application.
func (c *Controller) reconcileApp(appCtx context.Context, app *app.Application, appConfigFile string, appCancel context.CancelFunc, syncChan chan struct{}) {
	defer c.wg.Done() // Decrement WaitGroup counter when the goroutine finishes
	// Ensure the app's cancel func is removed from the map when this goroutine exits
	defer func() {
		c.mu.Lock()
		// Only delete if this goroutine was the one registered in runningApps
		if rt, ok := c.runningApps[app.Name]; ok && &rt.cancel == &appCancel {
			delete(c.runningApps, app.Name)
			c.logger.Debug("Removed app from runningApps map", zap.String("app", app.Name))
		}
		c.mu.Unlock()
		appCancel() // Also call the app's cancel func to ensure its context is marked done
	}()

	logger := c.logger.With(zap.String("app", app.Name))
	logger.Info("Starting reconciliation loop for application",
		zap.String("repo", app.RepoURL),
		zap.String("branch", app.Branch),
		zap.String("path", app.Path),
		zap.Duration("interval", app.PollingInterval))

	// Get cluster configuration for this application
	c.clusters.RLock()
	defer c.clusters.RUnlock()
	targetCluster, exists := c.clusters.Get(app.ClusterName)
	if !exists {
		logger.Error("Cluster configuration not found for application", zap.String("cluster", app.ClusterName))
		app.Status = "Error"
		app.Message = fmt.Sprintf("Cluster '%s' does not exist", app.ClusterName)
		app.ConsecutiveFailures = 0               // Reset failures on critical error
		c.saveAppStatus(app, appConfigFile, true) // Force save on critical error
		return
	}

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

	// Use kubeconfig path from the cluster configuration
	k8sClient, err := k8s.NewClientSet(logger, targetCluster.KubeconfigPath)
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
	logger.Info("Checking connectivity to Kubernetes cluster", zap.String("kubeconfig", targetCluster.KubeconfigPath))
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

		case <-syncChan: // Manual sync trigger
			logger.Info("Manual sync triggered via API for application.", zap.String("app", app.Name))
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
	previousFailures := app.ConsecutiveFailures

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
		if app.Status == "Error" || app.Status == "Pending" || app.Status == "SyncRequested" {
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

	c.saveAppStatus(app, appConfigFile, previousStatus != app.Status || previousHash != app.LastSyncedGitHash || previousFailures != app.ConsecutiveFailures)
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
		c.logger.Debug("No significant change to application status or failures, skipping save",
			zap.String("app", appToSave.Name),
			zap.String("current_status", appToSave.Status),
			zap.String("current_hash", appToSave.LastSyncedGitHash),
			zap.Int("current_failures", appToSave.ConsecutiveFailures))
	}
}
