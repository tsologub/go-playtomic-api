package client

import (
	"encoding/json"
	"fmt"
)

// APIError represents an error returned by the Playtomic API
type APIError struct {
	StatusCode int
	Message    string
	Details    map[string]interface{}
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// parseAPIError builds an APIError from a non-2xx response body, handling
// both the legacy error shape ({"error": "...", "details": {...}}) and the
// current shape used behind api.app.playtomic.io
// ({"status": "...", "localized_message": "..."}).
func parseAPIError(statusCode int, body []byte) *APIError {
	var legacy struct {
		Error   string                 `json:"error"`
		Details map[string]interface{} `json:"details"`
	}
	if err := json.Unmarshal(body, &legacy); err == nil && legacy.Error != "" {
		return &APIError{StatusCode: statusCode, Message: legacy.Error, Details: legacy.Details}
	}

	var modern struct {
		Status           string `json:"status"`
		LocalizedMessage string `json:"localized_message"`
	}
	if err := json.Unmarshal(body, &modern); err == nil && (modern.Status != "" || modern.LocalizedMessage != "") {
		message := modern.LocalizedMessage
		if message == "" {
			message = modern.Status
		}
		var details map[string]interface{}
		if modern.Status != "" {
			details = map[string]interface{}{"status": modern.Status}
		}
		return &APIError{StatusCode: statusCode, Message: message, Details: details}
	}

	return &APIError{StatusCode: statusCode, Message: "Unexpected response from API"}
}
