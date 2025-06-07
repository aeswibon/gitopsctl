package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"github.com/spf13/cobra"
)

// statusCmd represents the 'status' command which displays the status of registered GitOps applications.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of registered GitOps applications",
	Long:  `Displays the current status, last synced commit, and messages for all registered GitOps applications.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load applications from the configuration file
		applications, err := app.LoadApplications(app.DefaultAppConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load applications: %w", err)
		}

		// Check if there are no registered applications
		if len(applications.List()) == 0 {
			logger.Info("No applications registered.")
			return nil
		}

		// Create a tabwriter for formatted output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "NAME\tREPO URL\tPATH\tSTATUS\tLAST SYNCED HASH\tMESSAGE")
		fmt.Fprintln(w, "----\t--------\t----\t------\t----------------\t-------")

		// Iterate over applications and display their details
		for _, application := range applications.List() {
			hash := application.LastSyncedGitHash
			if len(hash) > 7 {
				hash = hash[:7] // Truncate hash for display
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				application.Name,
				application.RepoURL,
				application.Path,
				application.Status,
				hash,
				application.Message,
			)
		}
		w.Flush()

		return nil
	},
}

func init() {
	// Add the status command to the root command
	rootCmd.AddCommand(statusCmd)
}
