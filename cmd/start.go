package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"aeswibon.com/github/gitopsctl/internal/controller"
	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the GitOps controller",
	Long:  `Starts the GitOps controller, which continuously watches registered Git repositories and applies manifests to Kubernetes clusters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load applications from the configuration file
		applications, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		// Check if there are no registered applications
		if len(applications.List()) == 0 {
			logger.Warn("No applications registered. Please use 'gitopsctl register' to add an application.")
			return nil
		}

		// Create a new controller instance
		ctrl := controller.NewController(logger, applications)

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start the controller
		if err := ctrl.Start(app.DefaultAppConfigFile); err != nil {
			return fmt.Errorf("failed to start controller: %w", err)
		}

		// Wait for an interrupt signal
		<-sigChan
		logger.Info("Received shutdown signal. Stopping controller...")
		ctrl.Stop() // Stop the controller gracefully
		logger.Info("Controller stopped gracefully.")
		return nil
	},
}

func init() {
	// Add the start command to the root command
	rootCmd.AddCommand(startCmd)
}
