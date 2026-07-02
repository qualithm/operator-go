package operator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production management API base URL.
const DefaultBaseURL = "https://api.qualithm.com"

// TokenPrefix is the required prefix for member API tokens (qmt_<selector>.<verifier>).
const TokenPrefix = "qmt_"

// httpDoer abstracts the subset of [*http.Client] the client uses. Tests
// inject a fake implementation.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client talks to the qualithm platform management API with a member API
// token. Construct it with [New]. A zero Client is not usable.
type Client struct {
	baseURL  string
	token    string
	http     httpDoer
	dryRun   bool
	recorder func(Action)
}

// Option configures a [Client].
type Option func(*Client)

// WithBaseURL overrides the API base URL. Trailing slashes are trimmed.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		if u != "" {
			c.baseURL = strings.TrimRight(u, "/")
		}
	}
}

// WithHTTPClient sets the underlying HTTP client. When nil, a client with a
// 30-second timeout is used.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.http = hc
		}
	}
}

// WithDryRun enables dry-run mode: mutating requests are not sent and a
// [*DryRunError] carrying the planned [Action] is returned instead.
func WithDryRun(dry bool) Option {
	return func(c *Client) { c.dryRun = dry }
}

// WithRecorder registers a hook invoked with the [Action] of every mutating
// request just before it is sent (or would be sent, in dry-run mode).
func WithRecorder(fn func(Action)) Option {
	return func(c *Client) { c.recorder = fn }
}

// New constructs a [Client] authenticating with the given API token.
func New(token string, opts ...Option) (*Client, error) {
	if token == "" {
		return nil, errors.New("operator: empty API token")
	}
	if !strings.HasPrefix(token, TokenPrefix) {
		return nil, fmt.Errorf("operator: API token must start with %q", TokenPrefix)
	}
	c := &Client{
		baseURL: DefaultBaseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	if _, err := url.Parse(c.baseURL); err != nil {
		return nil, fmt.Errorf("operator: bad base URL %q: %w", c.baseURL, err)
	}
	return c, nil
}

// DryRun reports whether the client is in dry-run mode.
func (c *Client) DryRun() bool { return c.dryRun }

// envelope is the standard API response wrapper.
type envelope struct {
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
	Status  int             `json:"status,omitempty"`
}

func (c *Client) mutating(method string) bool {
	return method != http.MethodGet
}

// do issues an HTTP request and decodes the envelope's data field into out
// (when out is non-nil). Mutating requests in dry-run mode short-circuit with
// a [*DryRunError].
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if c.mutating(method) {
		action := Action{Method: method, Path: path, Body: body}
		if c.recorder != nil {
			c.recorder(action)
		}
		if c.dryRun {
			return &DryRunError{Action: action}
		}
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("operator: encode request body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("operator: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("operator: %s %s: %w", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("operator: read response: %w", err)
	}

	var env envelope
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			// Non-JSON body: fall back to the raw text as the message.
			env.Message = strings.TrimSpace(string(raw))
		}
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &ClientError{
			Method:     method,
			Path:       pathOnly(path),
			StatusCode: res.StatusCode,
			Message:    env.Message,
		}
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("operator: decode response: %w", err)
		}
	}
	return nil
}

func pathOnly(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i]
	}
	return path
}

// pageQuery renders page/limit query parameters, omitting zero values.
func pageQuery(page, limit int) string {
	v := url.Values{}
	if page > 0 {
		v.Set("page", fmt.Sprint(page))
	}
	if limit > 0 {
		v.Set("limit", fmt.Sprint(limit))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}
