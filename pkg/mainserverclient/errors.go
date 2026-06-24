package mainserverclient

import "fmt"

// APIError is returned when the main-server rejects or fails a request.
type APIError struct {
	Path       string
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}
	return fmt.Sprintf("main server %s: %d (%s)", e.Path, e.StatusCode, e.Message)
}

// IsAuthError reports 401 unauthorized (invalid token).
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*APIError); ok {
		return ae.StatusCode == 401
	}
	return false
}

// IsRetryable reports transient failures worth queueing.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*APIError); ok {
		return ae.StatusCode >= 500 || ae.StatusCode == 429
	}
	// network errors
	return true
}
