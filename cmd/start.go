package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aeswibon.com/github/gitopsctl/internal/api"
	"aeswibon.com/github/gitopsctl/internal/controller"
	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var apiAddress string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the GitOps controller and API server",
	Long: `Starts the GitOps controller, which continuously watches registered Git repositories and applies manifests to Kubernetes clusters.
Optionally starts a REST API server for programmatic management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load applications from the configuration file
		apps, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		// Check if there are no registered applications
		if len(apps.List()) == 0 {
			logger.Warn("No applications registered. Please use 'gitopsctl register' to add an application.")
		}

		// Create a new controller instance
		ctrl := controller.NewController(logger, apps)
		apiServer := api.NewServer(logger, apps, ctrl)

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start controller and API server in a goroutine
		go func() {
			if err := ctrl.Start(app.DefaultAppConfigFile); err != nil {
				logger.Fatal("Failed to start controller", zap.Error(err))
			}
		}()

		go func() {
			if err := apiServer.Start(apiAddress); err != nil && err != http.ErrServerClosed {
				logger.Fatal("Failed to start API server", zap.Error(err))
			}
		}()

		// Wait for an interrupt signal
		<-sigChan
		logger.Info("Received shutdown signal. Stopping controller...")

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Attempt to gracefully shut down the API server
		if err := apiServer.Stop(timeoutCtx); err != nil {
			logger.Error("API server shutdown error", zap.Error(err))
		}

		// Stop controller loops
		ctrl.Stop()

		logger.Info("Controller stopped gracefully.")
		return nil
	},
}

func init() {
	// Add the start command to the root command
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&apiAddress, "api-address", "a", ":8080", "Address for the API server to listen on (e.g., :8080, 0.0.0.0:8080)")

}
