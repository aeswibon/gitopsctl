package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/spf13/cobra"
)

var listClustersCmd = &cobra.Command{
	Use:   "list-clusters",
	Short: "List all registered Kubernetes clusters",
	Long:  `Displays the name, kubeconfig path, and status of all Kubernetes clusters registered with gitopsctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusters, err := cluster.LoadClusters(cluster.DefaultClusterConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load cluster configurations: %w", err)
		}

		if len(clusters.List()) == 0 {
			logger.Info("No clusters registered. Use 'gitopsctl register-cluster' to add one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "NAME\tKUBECONFIG PATH\tSTATUS\tMESSAGE\tREGISTERED AT")
		fmt.Fprintln(w, "----\t---------------\t------\t-------\t-------------")

		for _, cl := range clusters.List() {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				cl.Name,
				cl.KubeconfigPath,
				cl.Status,
				cl.Message,
				cl.RegisteredAt.Format("2006-01-02 15:04:05"),
			)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listClustersCmd)
}
