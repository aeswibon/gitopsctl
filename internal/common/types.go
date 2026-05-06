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
