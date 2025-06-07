package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"aeswibon.com/github/gitopsctl/internal/common"
	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// Flags for the register command
	appName        string // Name of the application
	repoURL        string // Git repository URL
	branch         string // Branch in the repository (optional, default is "master")
	pathInRepo     string // Path to Kubernetes manifests in the repository
	kubeconfigPath string // Path to the kubeconfig file for the target cluster
	interval       string // Polling interval for Git repository
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new GitOps application",
	Long: `Registers a new application to be managed by the GitOps controller.
This command defines where the Kubernetes manifests are located in Git
and which Kubernetes cluster they should be applied to.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if appName == "" {
			return fmt.Errorf("application name (--name) is required")
		}
		if repoURL == "" {
			return fmt.Errorf("repository URL (--repo) is required")
		}
		if pathInRepo == "" {
			return fmt.Errorf("path within repository (--path) is required")
		}
		if kubeconfigPath == "" {
			return fmt.Errorf("kubeconfig path (--kubeconfig) is required")
		}
		if interval == "" {
			return fmt.Errorf("polling interval (--interval) is required")
		}

		// Validate the format of the repository URL
		if !common.IsValidGitURL(repoURL) {
			return fmt.Errorf("invalid repository URL format: %s. Must be a valid HTTPS or SSH Git URL.", repoURL)
		}

		// Validate pathInRepo (simple check: no leading/trailing slashes, not empty after trim)
		pathInRepo = strings.TrimPrefix(strings.TrimSuffix(pathInRepo, "/"), "/")
		if pathInRepo == "" {
			return fmt.Errorf("invalid path within repository (--path): cannot be empty or just slashes")
		}

		info, err := os.Stat(kubeconfigPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file not found at '%s'", kubeconfigPath)
		}
		if err != nil {
			return fmt.Errorf("error accessing kubeconfig path '%s': %w", kubeconfigPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("kubeconfig path '%s' is a directory, not a file", kubeconfigPath)
		}
		// Basic check for read permissions (more robust permissions check might need syscalls or higher privileges)
		if file, err := os.Open(kubeconfigPath); err != nil {
			return fmt.Errorf("kubeconfig file at '%s' is not readable: %w", kubeconfigPath, err)
		} else {
			file.Close()
		}

		// Validate interval
		parsedInterval, err := time.ParseDuration(interval)
		if err != nil {
			return fmt.Errorf("invalid interval format: %w. Examples: 5m, 30s, 1h", err)
		}
		if parsedInterval <= 0 {
			return fmt.Errorf("interval must be a positive duration")
		}

		// Load existing applications from the configuration file
		apps, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		apps.Lock()
		defer apps.Unlock()

		// Check if the application already exists
		if _, exists := apps.Get(appName); exists {
			logger.Warn("Application with this name already exists. Updating it.", zap.String("name", appName))
		}

		// Create a new application object
		newApp := &app.Application{
			Name:                appName,
			RepoURL:             repoURL,
			Branch:              branch, // NEW: Assign branch
			Path:                pathInRepo,
			KubeconfigPath:      kubeconfigPath,
			Interval:            interval,
			PollingInterval:     parsedInterval,
			Status:              "Pending",
			Message:             "Application registered, awaiting first sync.",
			ConsecutiveFailures: 0, // NEW: Initialize failures
		}

		// Add the new application to the list
		apps.Add(newApp)
		if err := app.SaveApplications(apps, app.DefaultAppConfigFile); err != nil {
			return fmt.Errorf("failed to save application: %w", err)
		}

		// Log the successful registration of the application
		logger.Info("Application registered successfully!",
			zap.String("name", newApp.Name),
			zap.String("repo", newApp.RepoURL),
			zap.String("branch", newApp.Branch), // NEW: Log branch
			zap.String("path", newApp.Path),
			zap.String("kubeconfig", newApp.KubeconfigPath),
			zap.String("interval", newApp.Interval),
		)

		return nil
	},
}

func init() {
	// Add the register command to the root command
	rootCmd.AddCommand(registerCmd)

	// Define flags for the register command
	registerCmd.Flags().StringVarP(&appName, "name", "n", "", "Unique name for the application")
	registerCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Git repository URL (e.g., https://github.com/user/repo.git or git@github.com:user/repo.git)")
	registerCmd.Flags().StringVarP(&branch, "branch", "b", "master", "Branch in the repository (default is 'master')")
	registerCmd.Flags().StringVarP(&pathInRepo, "path", "p", "", "Path within the repository to Kubernetes manifests (e.g., 'k8s/prod')")
	registerCmd.Flags().StringVarP(&kubeconfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file for the target cluster")
	registerCmd.Flags().StringVarP(&interval, "interval", "i", "5m", "Polling interval for Git repository (e.g., '30s', '5m', '1h')")
}
