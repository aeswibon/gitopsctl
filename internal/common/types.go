package common

// ErrorResponse defines a standard error response structure.
// This structure is used in the API responses to provide consistent error messages and details.
// It includes a message field for the error description and an optional details field for additional context.
type ErrorResponse struct {
	// Message is a brief description of the error that occurred.
	Message string `json:"message"`
	// Details provides additional context or information about the error, if available.
	Details string `json:"details,omitempty"`
}

// SyncTriggerRequest defines the payload for triggering a manual sync.
// // This structure is used in the API requests to initiate a manual sync of an application with the GitOps controller.
// It currently does not require any specific fields, but can be extended in the future if needed.
type SyncTriggerRequest struct {
	// Currently empty, but can be extended with options like 'force' or 'commitHash'
}

// SyncTriggerResponse defines the response after triggering a manual sync.
// This structure is used in the API responses to indicate the result of a manual sync request.
// It includes a message field for confirmation and a status field to indicate the outcome of the sync operation.
type SyncTriggerResponse struct {
	// Message provides confirmation or details about the sync operation.
	Message string `json:"message"`
	// Status indicates the result of the sync operation (e.g., "success", "failed").
	Status string `json:"status"`
}

// Helper struct for API context (Echo's Context already exists)
// We might add custom context if needed, but for now, we'll pass the controller.
type APIContext struct {
	// Potentially hold references to controller, logger, etc.
	// For Echo, handlers usually receive echo.Context and you can store data in it.
}
