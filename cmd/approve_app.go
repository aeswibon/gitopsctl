package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aeswibon.com/github/gitopsctl/internal/events"
	"github.com/spf13/cobra"
)

var (
	approveAppName    string
	approveCommitHash string
)

var approveAppCmd = &cobra.Command{
	Use:     "approve-app",
	GroupID: "appGroup",
	Short:   "Approve a pending sync for an application (via controller API)",
	Long: `Sends an HTTP request to a running gitopsctl controller to approve a specific commit for an application.
This is used when the application's sync policy is set to 'manual'.`,
	Example: `
  gitopsctl approve-app -n myapp -c 827e3ef
  gitopsctl --api-url http://127.0.0.1:9090 approve-app -n myapp -c 827e3ef`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		base := strings.TrimRight(apiBaseURL, "/")
		path := fmt.Sprintf("%s/api/v1/applications/%s/approve", base, url.PathEscape(approveAppName))

		payload := map[string]string{
			"commitHash": approveCommitHash,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if apiAuthKey != "" {
			req.Header.Set("X-API-Key", apiAuthKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusAccepted:
			emitCommandEvent(events.TypeAppSyncRequested, map[string]any{
				"app":     approveAppName,
				"commit":  approveCommitHash,
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
	approveAppCmd.Flags().StringVarP(&approveAppName, "name", "n", "", "Application name (required)")
	approveAppCmd.Flags().StringVarP(&approveCommitHash, "commit", "c", "", "Commit hash to approve (required)")
	_ = approveAppCmd.MarkFlagRequired("name")
	_ = approveAppCmd.MarkFlagRequired("commit")
	rootCmd.AddCommand(approveAppCmd)
}
