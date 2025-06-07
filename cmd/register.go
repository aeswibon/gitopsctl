package cmd

import (
	"fmt"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	appName        string
	repoURL        string
	pathInRepo     string
	kubeconfigPath string
	interval       string
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new GitOps application",
	Long: `Registers a new application to be managed by the GitOps controller.
This command defines where the Kubernetes manifests are located in Git
and which Kubernetes cluster they should be applied to.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if appName == "" || repoURL == "" || pathInRepo == "" || kubeconfigPath == "" || interval == "" {
			return fmt.Errorf("all flags (--name, --repo, --path, --kubeconfig, --interval) are required")
		}

		parsedInterval, err := time.ParseDuration(interval)
		if err != nil {
			return fmt.Errorf("invalid interval format: %w. Examples: 5m, 30s, 1h", err)
		}

		applications, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		if _, exists := applications.Get(appName); exists {
			logger.Warn("Application with this name already exists. Updating it.", zap.String("name", appName))
		}

		newApp := &app.Application{
			Name:            appName,
			RepoURL:         repoURL,
			Path:            pathInRepo,
			KubeconfigPath:  kubeconfigPath,
			Interval:        interval,
			PollingInterval: parsedInterval, // Store parsed duration
			Status:          "Pending",
			Message:         "Application registered, awaiting first sync.",
		}

		applications.Add(newApp)

		if err := app.SaveApplications(applications, app.DefaultAppConfigFile); err != nil {
			return fmt.Errorf("failed to save application: %w", err)
		}

		logger.Info("Application registered successfully!",
			zap.String("name", newApp.Name),
			zap.String("repo", newApp.RepoURL),
			zap.String("path", newApp.Path),
			zap.String("kubeconfig", newApp.KubeconfigPath),
			zap.String("interval", newApp.Interval),
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)

	registerCmd.Flags().StringVarP(&appName, "name", "n", "", "Unique name for the application")
	registerCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Git repository URL (e.g., https://github.com/user/repo.git or git@github.com:user/repo.git)")
	registerCmd.Flags().StringVarP(&pathInRepo, "path", "p", "", "Path within the repository to Kubernetes manifests (e.g., 'k8s/prod')")
	registerCmd.Flags().StringVarP(&kubeconfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file for the target cluster")
	registerCmd.Flags().StringVarP(&interval, "interval", "i", "5m", "Polling interval for Git repository (e.g., '30s', '5m', '1h')")

	// Mark required flags
	registerCmd.MarkFlagRequired("name")
	registerCmd.MarkFlagRequired("repo")
	registerCmd.MarkFlagRequired("path")
	registerCmd.MarkFlagRequired("kubeconfig")
}
