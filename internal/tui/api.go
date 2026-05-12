package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AppResponse matches the JSON returned by GET /api/v1/applications
type AppResponse struct {
	Name                string       `json:"name"`
	RepoURL             string       `json:"repo_url"`
	Branch              string       `json:"branch"`
	Path                string       `json:"path"`
	ClusterName         string       `json:"cluster_name"`
	Interval            string       `json:"interval"`
	LastSyncedGitHash   string       `json:"last_synced_git_hash"`
	LatestGitHash       string       `json:"latest_git_hash"`
	Status              string       `json:"status"`
	Message             string       `json:"message"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	SyncPolicy          string       `json:"sync_policy"`
	ApprovedGitHash     string       `json:"approved_git_hash"`
	MaxRetries          int          `json:"max_retries"`
	InitialBackoff      string       `json:"initial_backoff"`
	MaxBackoff          string       `json:"max_backoff"`
	CreateNamespace     bool         `json:"create_namespace"`
	DependsOn           []string     `json:"depends_on"`
	Prune               bool         `json:"prune"`
	SyncWindows         []SyncWindow `json:"sync_windows"`
	WebhookURL          string       `json:"webhook_url"`
	WebhookSecret       string       `json:"webhook_secret"`
}

type SyncWindow struct {
	Start string   `json:"start"`
	End   string   `json:"end"`
	Days  []string `json:"days"`
	Deny  bool     `json:"deny"`
}

// ClusterResponse matches the JSON returned by GET /api/v1/clusters
type ClusterResponse struct {
	Name              string    `json:"name"`
	KubeconfigPath    string    `json:"kubeconfig_path"`
	RegisteredAt      time.Time `json:"registered_at"`
	Status            string    `json:"status"`
	Message           string    `json:"message"`
	LastCheckedAt     time.Time `json:"last_checked_at"`
	DefaultNamespace  string    `json:"default_namespace"`
	EnforceNamespace  bool      `json:"enforce_namespace"`
	AllowedNamespaces []string  `json:"allowed_namespaces"`
}

type apiClient struct {
	baseURL   string
	apiKey    string
	client    *http.Client
	sseClient *http.Client
}

func newAPIClient(baseURL, apiKey string) *apiClient {
	return &apiClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 10 * time.Second},
		sseClient: &http.Client{},
	}
}

func (c *apiClient) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	return req, nil
}

func (c *apiClient) getApplications() ([]AppResponse, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/applications", c.baseURL), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/clusters", c.baseURL), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
	req, err := c.newRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, name), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *apiClient) unregisterApp(name string) error {
	req, err := c.newRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/applications/%s", c.baseURL, name), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *apiClient) checkCluster(name string) error {
	req, err := c.newRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/clusters/%s/check", c.baseURL, name), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *apiClient) approveApp(name, commitHash string) error {
	payload, _ := json.Marshal(map[string]string{"commitHash": commitHash})
	req, err := c.newRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/applications/%s/approve", c.baseURL, name),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *apiClient) unregisterCluster(name string) error {
	req, err := c.newRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/clusters/%s", c.baseURL, name), nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// Event represents an integration event from the server.
type Event struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Source string         `json:"source"`
	Time   time.Time      `json:"time"`
	Data   map[string]any `json:"data"`
}

func (c *apiClient) getHistory() ([]Event, error) {
	req, err := c.newRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/events/history", c.baseURL), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var history []Event
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}
	return history, nil
}
