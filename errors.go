package operator

import "fmt"

// ClientError is returned when the management API answers with a non-2xx
// status. It is distinguishable from transport errors so callers can map
// status codes to stable CLI exit codes.
type ClientError struct {
	// Method is the HTTP method of the failed request.
	Method string
	// Path is the request path (no query string, no secrets).
	Path string
	// StatusCode is the HTTP status returned by the API.
	StatusCode int
	// Message is the human-readable message from the API envelope.
	Message string
}

// Error implements [error].
func (e *ClientError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("operator: %s %s: HTTP %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("operator: %s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
}

// Action describes a single mutating request the client would issue. It is
// carried by [DryRunError] and can be recorded via [WithRecorder].
type Action struct {
	// Method is the HTTP method (POST, PATCH, DELETE).
	Method string
	// Path is the request path, including any query string.
	Path string
	// Body is the JSON request body, or nil for bodyless requests.
	Body any
}

// DryRunError is returned by mutating methods when the client is in dry-run
// mode. It carries the [Action] that would have been performed so the caller
// can report the plan without applying it.
type DryRunError struct {
	Action Action
}

// Error implements [error].
func (e *DryRunError) Error() string {
	return fmt.Sprintf("dry-run: would %s %s", e.Action.Method, e.Action.Path)
}
