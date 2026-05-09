package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AppResponse matches the JSON returned by GET /api/v1/applications
type AppResponse struct {
	Name                string `json:"name"`
	RepoURL             string `json:"repo_url"`
	Branch              string `json:"branch"`
	Path                string `json:"path"`
	ClusterName         string `json:"cluster_name"`
	Interval            string `json:"interval"`
	LastSyncedGitHash   string `json:"last_synced_git_hash"`
	Status              string `json:"status"`
	Message             string `json:"message"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
}

// ClusterResponse matches the JSON returned by GET /api/v1/clusters
type ClusterResponse struct {
	Name           string    `json:"name"`
	KubeconfigPath string    `json:"kubeconfig_path"`
	RegisteredAt   time.Time `json:"registered_at"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	LastCheckedAt  time.Time `json:"last_checked_at"`
}

type apiClient struct {
	baseURL   string
	client    *http.Client
	sseClient *http.Client
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 10 * time.Second},
		sseClient: &http.Client{},
	}
}

func (c *apiClient) getApplications() ([]AppResponse, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/applications", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var apps []AppResponse
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *apiClient) getClusters() ([]ClusterResponse, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/clusters", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var clusters []ClusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return nil, err
	}
	return clusters, nil
}

func (c *apiClient) syncApp(name string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, name),
		"application/json", nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *apiClient) checkCluster(name string) error {
	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/v1/clusters/%s/check", c.baseURL, name),
		"application/json", nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
