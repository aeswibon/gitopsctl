package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"aeswibon.com/github/gitopsctl/internal/core/git"
	"aeswibon.com/github/gitopsctl/internal/core/k8s"
	"aeswibon.com/github/gitopsctl/internal/core/notifications"
	"aeswibon.com/github/gitopsctl/internal/events"
	"aeswibon.com/github/gitopsctl/internal/metrics"
	"go.uber.org/zap"
)

type ClusterCommandType string

type ClusterCommand struct {
	Type        ClusterCommandType
	ClusterName string
}

const (
	ClusterCommandCheck    ClusterCommandType = "CHECK"
	MaxConsecutiveFailures                    = 5
	BaseBackoffDuration                       = 5 * time.Second
	GitOperationTimeout                       = 60 * time.Second
	K8sApplyTimeout                           = 120 * time.Second
	K8sConnectTimeout                         = 10 * time.Second
	ConfigWatchInterval                       = 5 * time.Second
)

type AppCommandType string

const (
	AppCommandStart   AppCommandType = "START"
	AppCommandStop    AppCommandType = "STOP"
	AppCommandSync    AppCommandType = "SYNC"
	AppCommandApprove AppCommandType = "APPROVE"
)

type AppCommand struct {
	Type    AppCommandType
	AppName string
	Data    map[string]any
}

type appRuntime struct {
	cancel   context.CancelFunc
	syncChan chan struct{}
}

type Controller struct {
	logger             *zap.Logger
	apps               *app.Applications
	clusters           *cluster.Clusters
	ctx                context.Context
	cancel             context.CancelFunc
	appCommandChan     chan AppCommand
	clusterCommandChan chan ClusterCommand
	runningApps        map[string]*appRuntime
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	emitter            events.Emitter
	appConfigPath      string
}

type Option func(*Controller)

func WithEmitter(e events.Emitter) Option {
	return func(c *Controller) {
		c.emitter = e
	}
}

func NewController(logger *zap.Logger, apps *app.Applications, clusters *cluster.Clusters, opts ...Option) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := &Controller{
		logger:             logger,
		apps:               apps,
		clusters:           clusters,
		ctx:                ctx,
		cancel:             cancel,
		appCommandChan:     make(chan AppCommand, 50),
		clusterCommandChan: make(chan ClusterCommand, 50),
		runningApps:        make(map[string]*appRuntime),
	}
	for _, o := range opts {
		o(ctrl)
	}
	return ctrl
}

func (c *Controller) emit(ctx context.Context, typ events.Type, data map[string]any) {
	if c.emitter == nil {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	c.emitter.Emit(ctx, typ, data)
}

func (c *Controller) Emit(typ events.Type, data map[string]any) {
	c.emit(context.Background(), typ, data)
}

func (c *Controller) Start(appConfigFile string) error {
	c.appConfigPath = appConfigFile
	c.logger.Info("Starting GitOps controller...")

	c.wg.Add(1)
	go c.commandDispatcher(appConfigFile)

	c.wg.Add(1)
	go c.clusterHealthChecker()

	c.wg.Add(1)
	go c.configWatcher(appConfigFile)

	c.apps.RLock()
	appsToStart := c.apps.List()
	c.apps.RUnlock()

	for _, application := range appsToStart {
		c.appCommandChan <- AppCommand{Type: AppCommandStart, AppName: application.Name}
	}

	c.clusters.RLock()
	clustersToCheck := c.clusters.List()
	c.clusters.RUnlock()

	for _, cl := range clustersToCheck {
		c.clusterCommandChan <- ClusterCommand{Type: ClusterCommandCheck, ClusterName: cl.Name}
	}

	c.logger.Info("Initial reconciliation dispatched.")
	c.emit(c.ctx, events.TypeControllerStarted, map[string]any{
		"applications": len(appsToStart),
		"clusters":     len(clustersToCheck),
	})
	return nil
}

func (c *Controller) Stop() {
	c.emit(context.Background(), events.TypeControllerStopping, map[string]any{})
	c.logger.Info("Stopping GitOps controller...")
	c.cancel()
	close(c.appCommandChan)
	close(c.clusterCommandChan)
	c.wg.Wait()
	c.logger.Info("GitOps controller stopped.")
}

func (c *Controller) configWatcher(filePath string) {
	defer c.wg.Done()
	ticker := time.NewTicker(ConfigWatchInterval)
	defer ticker.Stop()

	var lastMod time.Time
	if info, err := os.Stat(filePath); err == nil {
		lastMod = info.ModTime()
	}

	for {
		select {
		case <-ticker.C:
			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				c.logger.Info("Configuration change detected, reloading...", zap.String("file", filePath))
				lastMod = info.ModTime()
				c.reloadApplications(filePath)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Controller) reloadApplications(filePath string) {
	newApps, err := app.LoadApplications(filePath)
	if err != nil {
		c.logger.Error("Failed to reload applications", zap.Error(err))
		return
	}

	c.apps.Lock()
	defer c.apps.Unlock()

	for _, a := range newApps.List() {
		existing, ok := c.apps.Get(a.Name)
		if !ok || existing.RepoURL != a.RepoURL || existing.Branch != a.Branch || existing.Path != a.Path || existing.ClusterName != a.ClusterName || existing.Interval != a.Interval {
			c.logger.Info("Detected new or updated application configuration", zap.String("app", a.Name))
			c.apps.Add(a)
			c.appCommandChan <- AppCommand{Type: AppCommandStart, AppName: a.Name}
		}
	}

	currentNames := make(map[string]bool)
	for _, a := range newApps.List() {
		currentNames[a.Name] = true
	}
	for name := range c.apps.Apps {
		if !currentNames[name] {
			c.logger.Info("Detected application removal", zap.String("app", name))
			c.appCommandChan <- AppCommand{Type: AppCommandStop, AppName: name}
			delete(c.apps.Apps, name)
		}
	}
}

func (c *Controller) StartApp(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandStart, AppName: appName}
}

func (c *Controller) StopApp(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandStop, AppName: appName}
}

func (c *Controller) TriggerSync(appName string) {
	c.appCommandChan <- AppCommand{Type: AppCommandSync, AppName: appName}
}

func (c *Controller) ApproveSync(appName string, commitHash string) {
	c.appCommandChan <- AppCommand{
		Type:    AppCommandApprove,
		AppName: appName,
		Data:    map[string]any{"commitHash": commitHash},
	}
}

func (c *Controller) TriggerClusterHealthCheck(clusterName string) {
	c.clusterCommandChan <- ClusterCommand{Type: ClusterCommandCheck, ClusterName: clusterName}
}

func (c *Controller) commandDispatcher(appConfigFile string) {
	defer c.wg.Done()
	c.logger.Info("Starting controller command dispatcher...")

	for {
		select {
		case cmd, ok := <-c.appCommandChan:
			if !ok {
				c.stopAllAppGoroutines()
				return
			}
			c.handleAppCommand(cmd, appConfigFile)
		case <-c.ctx.Done():
			c.stopAllAppGoroutines()
			return
		}
	}
}

func (c *Controller) clusterHealthChecker() {
	defer c.wg.Done()
	c.logger.Info("Cluster health checker started.")

	ticker := time.NewTicker(cluster.DefaultClusterHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.clusters.RLock()
			clustersToCheck := c.clusters.List()
			c.clusters.RUnlock()
			for _, cl := range clustersToCheck {
				c.performClusterHealthCheck(c.ctx, cl)
			}
		case cmd, ok := <-c.clusterCommandChan:
			if !ok {
				return
			}
			if cmd.Type == ClusterCommandCheck {
				cl, exists := c.clusters.Get(cmd.ClusterName)
				if exists {
					c.performClusterHealthCheck(c.ctx, cl)
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Controller) performClusterHealthCheck(ctx context.Context, cl *cluster.Cluster) {
	logger := c.logger.With(zap.String("cluster", cl.Name))
	k8sClient, err := k8s.NewClientSet(logger, cl.KubeconfigPath)
	if err != nil {
		cl.Status = "Error"
		cl.Message = fmt.Sprintf("Failed to create K8s client: %v", err)
	} else {
		checkCtx, checkCancel := context.WithTimeout(ctx, K8sConnectTimeout)
		defer checkCancel()
		if err := k8sClient.CheckConnectivity(checkCtx); err != nil {
			cl.Status = "Unreachable"
			cl.Message = fmt.Sprintf("Connectivity failed: %v", err)
			metrics.ClusterStatus.WithLabelValues(cl.Name).Set(0)
		} else {
			cl.Status = "Active"
			cl.Message = "Connectivity successful."
			metrics.ClusterStatus.WithLabelValues(cl.Name).Set(1)
		}
	}
	cl.LastCheckedAt = time.Now()

	c.clusters.Lock()
	_ = cluster.SaveClusters(c.clusters, cluster.DefaultClusterConfigFile)
	c.clusters.Unlock()

	c.emit(ctx, events.TypeClusterHealthCompleted, map[string]any{
		"cluster": cl.Name,
		"status":  cl.Status,
		"message": cl.Message,
	})
}

func (c *Controller) handleAppCommand(cmd AppCommand, appConfigFile string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch cmd.Type {
	case AppCommandStart:
		c.apps.RLock()
		appConfig, exists := c.apps.Get(cmd.AppName)
		if !exists {
			c.apps.RUnlock()
			return
		}

		c.clusters.RLock()
		_, clusterExists := c.clusters.Get(appConfig.ClusterName)
		c.clusters.RUnlock()

		if !clusterExists && appConfig.ClusterName != "" {
			c.apps.RUnlock()
			appConfig.Status = "Error"
			appConfig.Message = fmt.Sprintf("Cluster '%s' does not exist", appConfig.ClusterName)
			c.saveAppStatus(appConfig, appConfigFile, true)
			return
		}

		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			runtime.cancel()
		}

		appCopy := *appConfig
		c.apps.RUnlock()

		appCtx, appCancel := context.WithCancel(c.ctx)
		syncChan := make(chan struct{}, 1)

		c.wg.Add(1)
		c.runningApps[cmd.AppName] = &appRuntime{cancel: appCancel, syncChan: syncChan}
		go c.reconcileApp(appCtx, &appCopy, appConfigFile, appCancel, syncChan)

	case AppCommandStop:
		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			runtime.cancel()
		}

	case AppCommandSync:
		if runtime, ok := c.runningApps[cmd.AppName]; ok {
			select {
			case runtime.syncChan <- struct{}{}:
			default:
			}
		}

	case AppCommandApprove:
		c.apps.Lock()
		appConfig, exists := c.apps.Get(cmd.AppName)
		if !exists {
			c.apps.Unlock()
			return
		}
		commitHash, _ := cmd.Data["commitHash"].(string)
		appConfig.ApprovedGitHash = commitHash
		_ = app.SaveApplications(c.apps, appConfigFile)
		c.apps.Unlock()
		c.TriggerSync(cmd.AppName)
	}
}

func (c *Controller) stopAllAppGoroutines() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, runtime := range c.runningApps {
		runtime.cancel()
	}
}

func (c *Controller) reconcileApp(appCtx context.Context, app *app.Application, appConfigFile string, appCancel context.CancelFunc, syncChan chan struct{}) {
	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		if rt, ok := c.runningApps[app.Name]; ok && &rt.cancel == &appCancel {
			delete(c.runningApps, app.Name)
		}
		c.mu.Unlock()
		appCancel()
	}()

	logger := c.logger.With(zap.String("app", app.Name))

	if app.ClusterName == "" {
		app.Status = "Pending"
		app.Message = "Awaiting cluster assignment"
		c.saveAppStatus(app, appConfigFile, true)
	}

	repoDir, err := git.CreateTempRepoDir()
	if err != nil {
		app.Status = "Error"
		app.Message = fmt.Sprintf("Failed to create temp dir: %v", err)
		c.saveAppStatus(app, appConfigFile, true)
		return
	}
	defer func() { _ = git.CleanUpRepo(logger, repoDir) }()

	c.clusters.RLock()
	targetCluster, clusterExists := c.clusters.Get(app.ClusterName)
	c.clusters.RUnlock()

	var k8sClient *k8s.ClientSet
	if clusterExists {
		k8sClient, _ = k8s.NewClientSet(logger, targetCluster.KubeconfigPath)
	}

	interval := app.PollingInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.performSync(appCtx, logger, app, repoDir, k8sClient, appConfigFile, "initial")

	for {
		select {
		case <-ticker.C:
			c.performSync(appCtx, logger, app, repoDir, k8sClient, appConfigFile, "poll")
		case <-syncChan:
			c.performSync(appCtx, logger, app, repoDir, k8sClient, appConfigFile, "manual")
		case <-appCtx.Done():
			if app.Status != "Stopped" && app.Status != "Error" {
				app.Status = "Stopped"
				app.Message = "Controller shut down"
				c.saveAppStatus(app, appConfigFile, true)
			}
			return
		}
	}
}

func (c *Controller) performSync(ctx context.Context, logger *zap.Logger, app *app.Application, repoDir string, k8sClient *k8s.ClientSet, appConfigFile string, trigger string) {
	c.clusters.RLock()
	targetCluster, exists := c.clusters.Get(app.ClusterName)
	c.clusters.RUnlock()

	if !exists {
		app.Status = "Error"
		app.Message = fmt.Sprintf("Cluster '%s' does not exist", app.ClusterName)
		c.saveAppStatus(app, appConfigFile, true)
		return
	}

	if k8sClient == nil {
		k8sClient, _ = k8s.NewClientSet(logger, targetCluster.KubeconfigPath)
	}

	c.emit(ctx, events.TypeAppSyncStarted, map[string]any{"app": app.Name, "trigger": trigger})

	currentHash, err := git.CloneOrPull(ctx, logger, app.RepoURL, app.Branch, repoDir)
	if err != nil {
		app.Status = "Error"
		app.Message = fmt.Sprintf("Git error: %v", err)
		app.ConsecutiveFailures++
		c.saveAppStatus(app, appConfigFile, true)
		return
	}

	if currentHash == app.LastSyncedGitHash && trigger == "poll" {
		return
	}

	if app.SyncPolicy == "manual" && currentHash != app.ApprovedGitHash {
		app.Status = "OutOfSync"
		app.Message = fmt.Sprintf("Manual sync required. Latest: %s, Approved: %s", currentHash, app.ApprovedGitHash)
		c.saveAppStatus(app, appConfigFile, true)
		return
	}

	manifestsDir := filepath.Join(repoDir, app.Path)
	applyErrors := k8sClient.ApplyManifests(ctx, manifestsDir)
	if len(applyErrors) > 0 {
		errMsg := fmt.Sprintf("Apply error: %v", applyErrors[0])
		app.Status = "Error"
		app.Message = errMsg
		app.ConsecutiveFailures++
	} else {
		app.LastSyncedGitHash = currentHash
		app.Status = "Synced"
		app.Message = fmt.Sprintf("Synced to %s", currentHash)
		app.ConsecutiveFailures = 0
	}

	c.notify(app)
	c.saveAppStatus(app, appConfigFile, true)
}

func (c *Controller) notify(app *app.Application) {
	if app.WebhookURL == "" {
		return
	}

	go notifications.SendWebhook(c.logger, app.WebhookURL, app.WebhookSecret, notifications.Notification{
		App:       app.Name,
		Cluster:   app.ClusterName,
		Status:    app.Status,
		Message:   app.Message,
		Commit:    app.LastSyncedGitHash,
		Timestamp: time.Now(),
	})
}

func (c *Controller) saveAppStatus(appToSave *app.Application, appConfigFile string, forceSave bool) {
	c.apps.Lock()
	defer c.apps.Unlock()

	originalApp, ok := c.apps.Apps[appToSave.Name]
	if !ok {
		return
	}

	originalApp.Status = appToSave.Status
	originalApp.Message = appToSave.Message
	originalApp.LastSyncedGitHash = appToSave.LastSyncedGitHash
	originalApp.ConsecutiveFailures = appToSave.ConsecutiveFailures

	_ = app.SaveApplications(c.apps, appConfigFile)
}
