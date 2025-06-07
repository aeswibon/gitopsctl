package cmd

import (
	"fmt"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	unregisterAppName string // Name of the application to unregister
)

var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Unregister a GitOps application",
	Long: `Removes a registered application from GitOps controller's management.
Note: This only removes the application from the controller's configuration
and does NOT delete any resources from the Kubernetes cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flag
		if unregisterAppName == "" {
			return fmt.Errorf("application name (--name) is required")
		}

		// Load applications from the configuration file
		applications, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		// Acquire lock before modifying and saving
		applications.Lock()
		defer applications.Unlock()

		// Check if the application exists
		_, exists := applications.Get(unregisterAppName)
		if !exists {
			logger.Warn("Application not found, nothing to unregister.", zap.String("name", unregisterAppName))
			return nil
		}

		// Remove the application from the list
		applications.Delete(unregisterAppName) // Use Delete method
		if err := app.SaveApplications(applications, app.DefaultAppConfigFile); err != nil {
			return fmt.Errorf("failed to save applications after unregister: %w", err)
		}

		// Log the successful unregistration of the application
		logger.Info("Application unregistered successfully!", zap.String("name", unregisterAppName))
		return nil
	},
}

func init() {
	// Add the unregister command to the root command
	rootCmd.AddCommand(unregisterCmd)

	// Define flags for the unregister command
	unregisterCmd.Flags().StringVarP(&unregisterAppName, "name", "n", "", "Name of the application to unregister")
	unregisterCmd.MarkFlagRequired("name") // Mark the name flag as required
}
