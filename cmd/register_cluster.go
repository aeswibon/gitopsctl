package cmd

import (
	"fmt"
	"os"
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	clusterRegName        string
	clusterKubeconfigPath string
)

var registerClusterCmd = &cobra.Command{
	Use:   "register-cluster",
	Short: "Register a new Kubernetes cluster",
	Long: `Registers a new Kubernetes cluster with gitopsctl.
This allows applications to reference clusters by name.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if clusterRegName == "" {
			return fmt.Errorf("cluster name (--name) is required")
		}
		if clusterKubeconfigPath == "" {
			return fmt.Errorf("kubeconfig path (--kubeconfig) is required")
		}

		// Validate kubeconfig file existence and readability
		info, err := os.Stat(clusterKubeconfigPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file not found at '%s'", clusterKubeconfigPath)
		}
		if err != nil {
			return fmt.Errorf("error accessing kubeconfig path '%s': %w", clusterKubeconfigPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("kubeconfig path '%s' is a directory, not a file", clusterKubeconfigPath)
		}
		if file, err := os.Open(clusterKubeconfigPath); err != nil {
			return fmt.Errorf("kubeconfig file at '%s' is not readable: %w", clusterKubeconfigPath, err)
		} else {
			file.Close()
		}

		clusters, err := cluster.LoadClusters(cluster.DefaultClusterConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load cluster configurations: %w", err)
		}

		clusters.Lock()
		defer clusters.Unlock()

		if _, exists := clusters.Get(clusterRegName); exists {
			logger.Warn("Cluster with this name already exists. Updating its kubeconfig.", zap.String("name", clusterRegName))
		}

		newCluster := &cluster.Cluster{
			Name:           clusterRegName,
			KubeconfigPath: clusterKubeconfigPath,
			RegisteredAt:   time.Now(),
			Status:         "Active",
			Message:        "Cluster registered successfully.",
		}

		clusters.Add(newCluster)

		if err := cluster.SaveClusters(clusters, cluster.DefaultClusterConfigFile); err != nil {
			return fmt.Errorf("failed to save cluster: %w", err)
		}

		logger.Info("Cluster registered successfully!",
			zap.String("name", newCluster.Name),
			zap.String("kubeconfig", newCluster.KubeconfigPath),
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerClusterCmd)

	registerClusterCmd.Flags().StringVarP(&clusterRegName, "name", "n", "", "Unique name for the Kubernetes cluster")
	registerClusterCmd.Flags().StringVarP(&clusterKubeconfigPath, "kubeconfig", "k", "", "Path to the kubeconfig file for this cluster")

	registerClusterCmd.MarkFlagRequired("name")
	registerClusterCmd.MarkFlagRequired("kubeconfig")
}
