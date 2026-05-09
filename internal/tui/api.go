package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
)

type apiClient struct {
	baseURL string
	client  *http.Client
	sseClient *http.Client
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		sseClient: &http.Client{},
	}
}

func (c *apiClient) getApplications() ([]appcore.Application, error) {
	// Fixed route: applications instead of apps
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/applications", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var apps []appcore.Application
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *apiClient) getClusters() ([]clustercore.Cluster, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/clusters", c.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var clusters []clustercore.Cluster
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return nil, err
	}
	return clusters, nil
}

func (c *apiClient) syncApp(name string) error {
	resp, err := c.client.Post(fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, name), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *apiClient) checkCluster(name string) error {
	resp, err := c.client.Post(fmt.Sprintf("%s/api/v1/clusters/%s/check", c.baseURL, name), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
