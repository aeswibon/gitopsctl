package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile string
	logger  *zap.Logger
)

var (
	// List Global flags
	outputFormat string // Output format: table, json, yaml
	showStatus   string // Filter by status: all, active, inactive, error
	sortBy       string // Sort by: name, status, registered
	showDetails  bool   // Show additional details
	noHeader     bool   // Hide table headers
)

var rootCmd = &cobra.Command{
	Use:   "gitopsctl",
	Short: "A lightweight GitOps controller for Kubernetes",
	Long: `gitopsctl is a minimalistic, self-hosted GitOps controller that watches Git repositories
and applies Kubernetes manifests to target clusters.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Zap logger
		// Create a new production configuration for the logger
		config := zap.NewProductionConfig()
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}

		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		config.EncoderConfig.CallerKey = "caller"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.TimeKey = "ts"
		config.EncoderConfig.MessageKey = "msg"

		config.Encoding = "console"
		config.DisableStacktrace = true

		var err error
		logger, err = config.Build() // Use the exported variable
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		zap.ReplaceGlobals(logger)
		return nil
	},
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if logger != nil {
			logger.Error("Command execution failed", zap.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// Logger returns the global zap logger instance.
func Logger() *zap.Logger {
	return logger
}

func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.AddGroup(appGroup)
	rootCmd.AddGroup(clusterGroup)
	rootCmd.AddCommand(startCmd)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gitopsctl.yaml)")
}
