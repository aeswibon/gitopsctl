package cmd

import (
	"fmt"

	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	clusterUnregName string
)

var unregisterClusterCmd = &cobra.Command{
	Use:   "unregister-cluster",
	Short: "Unregister a Kubernetes cluster",
	Long: `Removes a registered Kubernetes cluster from gitopsctl's management.
Note: This only removes the cluster from the controller's configuration.
Any applications associated with this cluster will become dysfunctional
until they are updated to reference a valid cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if clusterUnregName == "" {
			return fmt.Errorf("cluster name (--name) is required")
		}

		clusters, err := cluster.LoadClusters(cluster.DefaultClusterConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load cluster configurations: %w", err)
		}

		clusters.Lock()
		defer clusters.Unlock()

		_, exists := clusters.Get(clusterUnregName)
		if !exists {
			logger.Warn("Cluster not found, nothing to unregister.", zap.String("name", clusterUnregName))
			return nil
		}

		clusters.Delete(clusterUnregName)
		if err := cluster.SaveClusters(clusters, cluster.DefaultClusterConfigFile); err != nil {
			return fmt.Errorf("failed to save clusters after unregister: %w", err)
		}

		logger.Info("Cluster unregistered successfully!", zap.String("name", clusterUnregName))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unregisterClusterCmd)
	unregisterClusterCmd.Flags().StringVarP(&clusterUnregName, "name", "n", "", "Name of the cluster to unregister")
	unregisterClusterCmd.MarkFlagRequired("name")
}
