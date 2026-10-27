package operator

import (
	"fmt"
	"net/http"
	"time"
)

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
	// Code is the stable error code from the API envelope, when it sent one
	// (e.g. "backend_unavailable"). Branch on Code, never on Message.
	Code string
	// RetryAfter is how long the API asked the caller to wait before retrying,
	// from its Retry-After header; zero when absent.
	RetryAfter time.Duration
	// RequestID identifies the request in the API's logs: the envelope's
	// requestId, else the Request-Id header; empty when neither was sent.
	RequestID string
}

// Unavailable reports whether the API could not serve the request because a
// backend was briefly unavailable. The request is safe to retry after
// RetryAfter.
func (e *ClientError) Unavailable() bool {
	return e.StatusCode == 503
}

// Error implements [error].
func (e *ClientError) Error() string {
	msg := fmt.Sprintf("operator: %s %s: HTTP %d", e.Method, e.Path, e.StatusCode)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.RequestID != "" {
		msg += " (request " + e.RequestID + ")"
	}
	return msg
}

// requestID picks the envelope's requestId, falling back to the Request-Id header.
func requestID(env envelope, header http.Header) string {
	if env.RequestID != "" {
		return env.RequestID
	}
	return header.Get("Request-Id")
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
