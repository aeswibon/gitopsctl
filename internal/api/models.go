package api

import (
	"time"

	"aeswibon.com/github/gitopsctl/internal/core/app"
	"aeswibon.com/github/gitopsctl/internal/core/cluster"
)

// RegisterAppRequest defines the payload for creating/registering a new application.
//
// This structure is used in the API requests to register a new application with the GitOps controller.
type RegisterAppRequest struct {
	// Name is the unique identifier for the application.
	Name string `json:"name" validate:"required"`
	// RepoURL is the URL of the Git repository where the application's manifests are stored.
	RepoURL string `json:"repoURL" validate:"required,url"`
	// Branch is the branch in the Git repository that contains the application's manifests.
	Branch string `json:"branch" validate:"required"`
	// Path is the directory path within the repository where the manifests are located.
	Path string `json:"path" validate:"required"`
	// ClusterName is the name of the Kubernetes cluster where the application will be deployed.
	ClusterName string `json:"clusterName" validate:"required"`
	// Interval is the frequency at which the application should be synced with the Git repository.
	Interval string `json:"interval" validate:"required"`
}

// ApplicationResponse defines the structure for returning application details via the API.
//
// This structure is used in the API responses to provide information about registered applications.
// It is designed to be user-friendly and includes all necessary fields for the application's configuration and status.
type ApplicationResponse struct {
	// Name is the unique identifier for the application.
	Name string `json:"name"`
	// RepoURL is the URL of the Git repository where the application's manifests are stored.
	RepoURL string `json:"repoURL"`
	// Branch is the branch in the Git repository that contains the application's manifests.
	Branch string `json:"branch"`
	// Path is the directory path within the repository where the manifests are located.
	Path string `json:"path"`
	// ClusterName is the name of the Kubernetes cluster where the application will be deployed.
	ClusterName string `json:"clusterName"`
	// Interval is the frequency at which the application should be synced with the Git repository.
	Interval string `json:"interval"`
	// LastSyncedGitHash is the last commit hash that was successfully synced from the Git repository.
	LastSyncedGitHash string `json:"lastSyncedGitHash"`
	// Status indicates the current status of the application (e.g., "active", "inactive", "error").
	Status string `json:"status"`
	// Message provides additional information about the application's status, such as error messages or warnings.
	Message string `json:"message"`
	// ConsecutiveFailures counts the number of consecutive sync failures for the application.
	ConsecutiveFailures int `json:"consecutiveFailures"`
	// LastUpdated is the timestamp of the last update to the application's status.
	LastUpdated string `json:"lastUpdated"`
}

// RegisterClusterRequest defines the payload for registering a new cluster.
//
// This structure is used in the API requests to register a new Kubernetes cluster with the GitOps controller.
type RegisterClusterRequest struct {
	// Name is the unique identifier for the cluster.
	Name string `json:"name" validate:"required"`
	// KubeconfigPath is the file path to the kubeconfig file for accessing the Kubernetes cluster.
	KubeconfigPath string `json:"kubeconfigPath" validate:"required,kubeconfigfile"`
}

// ClusterResponse defines the structure for returning cluster details via the API.
//
// This structure is used in the API responses to provide information about registered clusters.
type ClusterResponse struct {
	// Name is the unique identifier for the cluster.
	Name string `json:"name"`
	// KubeconfigPath is the file path to the kubeconfig file for accessing the Kubernetes cluster.
	KubeconfigPath string `json:"kubeconfigPath"`
	// RegisteredAt is the timestamp when the cluster was registered with the GitOps controller.
	RegisteredAt string `json:"registeredAt"`
	// Status indicates the current status of the cluster (e.g., "active", "inactive", "error").
	Status string `json:"status"`
	// Message provides additional information about the cluster's status, such as error messages or warnings.
	Message string `json:"message"`
}

// ErrorResponse defines a standard error response structure.
//
// This structure is used in the API responses to provide consistent error messages and details.
// It includes a message field for the error description and an optional details field for additional context.
type ErrorResponse struct {
	// Message is a brief description of the error that occurred.
	Message string `json:"message"`
	// Details provides additional context or information about the error, if available.
	Details string `json:"details,omitempty"`
}

// SyncTriggerRequest defines the payload for triggering a manual sync.
//
// This structure is used in the API requests to initiate a manual sync of an application with the GitOps controller.
// It currently does not require any specific fields, but can be extended in the future if needed.
type SyncTriggerRequest struct {
	// Currently empty, but can be extended with options like 'force' or 'commitHash'
}

// SyncTriggerResponse defines the response after triggering a manual sync.
//
// This structure is used in the API responses to indicate the result of a manual sync request.
// It includes a message field for confirmation and a status field to indicate the outcome of the sync operation.
type SyncTriggerResponse struct {
	// Message provides confirmation or details about the sync operation.
	Message string `json:"message"`
	// Status indicates the result of the sync operation (e.g., "success", "failed").
	Status string `json:"status"`
}

// Helper struct for API context (Echo's Context already exists)
//
// We might add custom context if needed, but for now, we'll pass the controller.
type APIContext struct {
	// Potentially hold references to controller, logger, etc.
	// For Echo, handlers usually receive echo.Context and you can store data in it.
}

// ApplicationMap is a type alias for the applications map.
//
// This is used internally by the API to reference the controller's applications.
type ApplicationMap map[string]*ApplicationResponse

// ConvertAppToResponse converts an internal app.Application struct to an API-friendly ApplicationResponse.
//
// This function extracts relevant fields from the app.Application struct and formats them for API responses.
func ConvertAppToResponse(app *app.Application) ApplicationResponse {
	return ApplicationResponse{
		Name:                app.Name,
		RepoURL:             app.RepoURL,
		Branch:              app.Branch,
		Path:                app.Path,
		ClusterName:         app.ClusterName,
		Interval:            app.Interval,
		LastSyncedGitHash:   app.LastSyncedGitHash,
		Status:              app.Status,
		Message:             app.Message,
		ConsecutiveFailures: app.ConsecutiveFailures,
		LastUpdated:         time.Now().Format(time.RFC3339),
	}
}

// ConvertClusterToResponse converts an internal cluster.Cluster struct to an API-friendly ClusterResponse.
//
// This function extracts relevant fields from the cluster.Cluster struct and formats them for API responses.
func ConvertClusterToResponse(cl *cluster.Cluster) ClusterResponse {
	return ClusterResponse{
		Name:           cl.Name,
		KubeconfigPath: cl.KubeconfigPath,
		RegisteredAt:   cl.RegisteredAt.Format(time.RFC3339),
		Status:         cl.Status,
		Message:        cl.Message,
	}
}
