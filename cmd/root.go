package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile string      // Path to the configuration file
	logger  *zap.Logger // Global logger instance
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitopsctl",
	Short: "A lightweight GitOps controller for Kubernetes",
	Long: `gitopsctl is a minimalistic, self-hosted GitOps controller
that watches Git repositories and applies Kubernetes manifests
to target clusters.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Zap logger
		// Create a new production configuration for the logger
		config := zap.NewProductionConfig()
		config.OutputPaths = []string{"stdout"}                             // Log to stdout
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        // Use ISO8601 format for timestamps
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // For colored output
		config.Encoding = "console"                                         // Set the logger encoding to console
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)                  // Set default log level to Info

		// Build the logger instance
		var err error
		logger, err = config.Build()
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		zap.ReplaceGlobals(logger) // Set as global logger
		return nil
	},
	SilenceUsage: true, // Don't print usage on error
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Execute the root command and handle errors
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Define persistent flags and configuration settings
	// Cobra supports persistent flags, which, if defined here,
	// will be available to all subcommands.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gitopsctl.yaml)")
}
