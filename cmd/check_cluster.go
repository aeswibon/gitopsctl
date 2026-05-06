package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aeswibon.com/github/gitopsctl/internal/events"
	"github.com/spf13/cobra"
)

var checkClusterName string

var checkClusterCmd = &cobra.Command{
	Use:     "check-cluster",
	GroupID: "clusterGroup",
	Short:   "Request an immediate connectivity check for a cluster (via controller API)",
	Long: `Sends an HTTP request to a running gitopsctl controller to health-check one cluster.

Requires 'gitopsctl start' with API reachable (--api-url must match the controller).`,
	Example: `
  gitopsctl check-cluster -n production
  gitopsctl --api-url http://127.0.0.1:9090 check-cluster -n staging`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		base := strings.TrimRight(apiBaseURL, "/")
		path := fmt.Sprintf("%s/api/v1/clusters/%s/check", base, url.PathEscape(checkClusterName))

		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest(http.MethodPost, path, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusAccepted:
			emitCommandEvent(events.TypeClusterHealthCheckRequested, map[string]any{
				"cluster": checkClusterName,
				"api_url": apiBaseURL,
			})
			fmt.Println(strings.TrimSpace(string(body)))
			return nil
		default:
			return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
	},
}

func init() {
	checkClusterCmd.Flags().StringVarP(&checkClusterName, "name", "n", "", "Cluster name (required)")
	_ = checkClusterCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(checkClusterCmd)
}
