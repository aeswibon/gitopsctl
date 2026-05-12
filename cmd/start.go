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
	"aeswibon.com/github/gitopsctl/internal/events"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	apiAddress string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the GitOps controller and API server",
	Long: `Starts the GitOps controller, which continuously watches registered Git repositories and applies manifests to Kubernetes clusters,
and serves a REST API for automation.

Phase 2: optionally emit integration events (JSONL file and/or HTTP webhook) for external dashboards.`,
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
			logger.Warn("No applications registered. Use 'gitopsctl register-apps' to add an application.")
		}

		if len(clusters.List()) == 0 {
			logger.Warn("No clusters registered. Use 'gitopsctl register-cluster' to add a cluster.")
		}

		var ctrlOpts []controller.Option
		var sinks []events.Sink
		streamSink := events.NewStreamSink()
		historySink := events.NewHistorySink(100)
		sinks = append(sinks, streamSink, historySink) // Available for live stream and history.
		if eventsFile != "" {
			fs, err := events.NewFileSink(eventsFile)
			if err != nil {
				return fmt.Errorf("events file sink: %w", err)
			}
			sinks = append(sinks, fs)
		}
		if eventsWebhookURL != "" {
			sinks = append(sinks, events.NewWebhookSinkWithOptions(eventsWebhookURL, eventsWebhookBearer, events.WebhookOptions{
				Timeout:       eventsWebhookTimeout,
				Retries:       eventsWebhookRetry,
				RetryBackoff:  eventsWebhookBackoff,
				SigningSecret: eventsWebhookSecret,
			}))
		}
		bus := events.NewBus(logger, "gitopsctl-controller", sinks...)
		ctrlOpts = append(ctrlOpts, controller.WithEmitter(bus))
		logger.Info("Integration events enabled",
			zap.Int("sinks", len(sinks)),
			zap.Bool("sse_stream", true),
			zap.Bool("jsonl_sink", eventsFile != ""),
			zap.Bool("webhook_sink", eventsWebhookURL != ""))

		ctrl := controller.NewController(logger, apps, clusters, ctrlOpts...)
		apiServer := api.NewServer(logger, apps, clusters, ctrl, streamSink, historySink, apiAuthKey)

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
