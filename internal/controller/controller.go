package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/git"
	"aeswibon.com/github/gitopsctl/internal/core/k8s"
	"go.uber.org/zap"
)

// Controller orchestrates the GitOps reconciliation loop.
type Controller struct {
	logger *zap.Logger
	apps   *app.Applications
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // To wait for all goroutines to finish
}

// NewController creates a new Controller instance.
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
func (c *Controller) Start(appConfigFile string) error {
	c.logger.Info("Starting GitOps controller...")

	for _, application := range c.apps.List() {
		c.wg.Add(1)
		go c.reconcileApp(application, appConfigFile)
	}

	c.logger.Info("All application reconciliation loops started.")
	return nil
}

// Stop gracefully stops all reconciliation loops.
func (c *Controller) Stop() {
	c.logger.Info("Stopping GitOps controller...")
	c.cancel()  // Signal all goroutines to stop
	c.wg.Wait() // Wait for all goroutines to finish
	c.logger.Info("GitOps controller stopped.")
}

// reconcileApp runs the GitOps loop for a single application.
func (c *Controller) reconcileApp(application *app.Application, appConfigFile string) {
	defer c.wg.Done()

	logger := c.logger.With(zap.String("app", application.Name))
	logger.Info("Starting reconciliation loop for application",
		zap.String("repo", application.RepoURL),
		zap.String("path", application.Path),
		zap.Duration("interval", application.PollingInterval))

	// Create a temporary directory for this app's Git repo
	repoDir, err := git.CreateTempRepoDir()
	if err != nil {
		logger.Error("Failed to create temporary repo directory", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("Failed to create temp dir: %v", err)
		c.saveAppStatus(appConfigFile)
		return
	}
	defer func() {
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
		c.saveAppStatus(appConfigFile)
		return
	}

	// Initial connectivity check (optional but good practice)
	if err := k8sClient.CheckConnectivity(c.ctx); err != nil {
		logger.Error("Failed to connect to Kubernetes cluster", zap.Error(err))
		application.Status = "Error"
		application.Message = fmt.Sprintf("K8s connectivity error: %v", err)
		c.saveAppStatus(appConfigFile)
		return
	}

	ticker := time.NewTicker(application.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("Polling Git repository...")
			currentHash, err := git.CloneOrPull(logger, application.RepoURL, "master", repoDir) // Assuming 'master' branch for now
			if err != nil {
				logger.Error("Failed to pull Git repository", zap.Error(err))
				application.Status = "Error"
				application.Message = fmt.Sprintf("Git pull error: %v", err)
				c.saveAppStatus(appConfigFile)
				continue
			}

			if currentHash == application.LastSyncedGitHash {
				logger.Debug("No new changes detected in Git repository", zap.String("hash", currentHash))
				application.Status = "Synced" // Keep status as synced if no new changes and no errors
				application.Message = fmt.Sprintf("Up to date at %s", currentHash)
				// Even if no change, update timestamp for clarity, but only save if actual change to status or hash
				c.saveAppStatus(appConfigFile) // Save to update status if it changed from Error to Synced
				continue
			}

			logger.Info("New changes detected in Git repository",
				zap.String("oldHash", application.LastSyncedGitHash),
				zap.String("newHash", currentHash))

			manifestsDir := filepath.Join(repoDir, application.Path)
			if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
				logger.Error("Manifests path does not exist in repository", zap.String("path", application.Path))
				application.Status = "Error"
				application.Message = fmt.Sprintf("Manifests path %s not found in repo", application.Path)
				c.saveAppStatus(appConfigFile)
				continue
			}

			logger.Info("Applying Kubernetes manifests...", zap.String("sourceDir", manifestsDir))
			err = k8sClient.ApplyManifests(c.ctx, manifestsDir)
			if err != nil {
				logger.Error("Failed to apply Kubernetes manifests", zap.Error(err))
				application.Status = "Error"
				application.Message = fmt.Sprintf("K8s apply error: %v", err)
				c.saveAppStatus(appConfigFile)
				continue
			}

			application.LastSyncedGitHash = currentHash
			application.Status = "Synced"
			application.Message = fmt.Sprintf("Successfully synced to %s", currentHash)
			logger.Info("Successfully applied Kubernetes manifests", zap.String("hash", currentHash))

			c.saveAppStatus(appConfigFile)

		case <-c.ctx.Done():
			logger.Info("Reconciliation loop stopping for application.", zap.String("reason", c.ctx.Err().Error()))
			application.Status = "Stopped"
			application.Message = "Controller shut down."
			c.saveAppStatus(appConfigFile)
			return
		}
	}
}

// saveAppStatus is a helper to update and persist the application's status.
func (c *Controller) saveAppStatus(appConfigFile string) {
	c.apps.Lock()
	defer c.apps.Unlock()
	// No need to reload, just save the in-memory state which is updated by the goroutine
	if err := app.SaveApplications(c.apps, appConfigFile); err != nil {
		c.logger.Error("Failed to save application status to file", zap.Error(err))
	}
}
