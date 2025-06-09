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
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
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
		apps, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		clusters, err := cluster.LoadClusters(cluster.DefaultClusterConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load clusters: %w", err)
		}

		if len(apps.List()) == 0 {
			logger.Warn("No applications registered. Please use 'gitopsctl register' to add an application.")
		}

		if len(clusters.List()) == 0 {
			logger.Warn("No clusters registered. Please use 'gitopsctl register' to add a cluster.")
		}

		ctrl := controller.NewController(logger, apps, clusters)
		apiServer := api.NewServer(logger, apps, clusters, ctrl)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

		if err := apiServer.Stop(timeoutCtx); err != nil {
			logger.Error("API server shutdown error", zap.Error(err))
		}
		ctrl.Stop()

		logger.Info("Controller stopped gracefully.")
		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&apiAddress, "api-address", "a", ":8080", "Address for the API server to listen on (e.g., :8080, 0.0.0.0:8080)")

}
