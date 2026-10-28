package operator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the production management API base URL.
const DefaultBaseURL = "https://api.qualithm.com"

// TokenPrefix is the required prefix for member API tokens. A token is
// qmt_<selector>.<verifier>, and one issued with a checksum ends in eight more
// hex characters: the CRC-32 of everything before them.
const TokenPrefix = "qmt_"

// ErrTokenChecksum reports an API token whose checksum does not match the rest
// of it: the token was mistyped or cut short.
var ErrTokenChecksum = errors.New("operator: API token is mistyped or truncated")

const (
	verifierLen = 64
	checksumLen = 8
)

// checkToken rejects a token whose checksum is wrong. A token issued without a
// checksum has a shorter verifier and is left for the API to judge.
func checkToken(token string) error {
	_, verifier, found := strings.Cut(token, ".")
	if !found || len(verifier) != verifierLen+checksumLen || !isLowerHex(verifier) {
		return nil
	}
	split := len(token) - checksumLen
	if fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(token[:split]))) != token[split:] {
		return ErrTokenChecksum
	}
	return nil
}

func isLowerHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// httpDoer abstracts the subset of [*http.Client] the client uses. Tests
// inject a fake implementation.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client talks to the Qualithm platform management API with a member API
// token. Construct it with [New]. A zero Client is not usable.
type Client struct {
	baseURL   string
	token     string
	teamID    string
	teamMu    sync.Mutex
	http      httpDoer
	dryRun    bool
	recorder  func(Action)
	userAgent string
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

// WithTeam sets the team the client acts on. Without it, the client asks the API on first use:
// an API token is pinned to one team, so that is the team it finds.
func WithTeam(teamID string) Option {
	return func(c *Client) { c.teamID = teamID }
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

// WithUserAgent prepends a product token to the User-Agent header sent on every
// request, e.g. WithUserAgent("qualithm-mcp/1.2.3") sends
// "qualithm-mcp/1.2.3 operator-go/<Version>", so the API can tell which tool
// made a call and which library version it used. Empty values are ignored,
// leaving just "operator-go/<Version>".
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// New constructs a [Client] authenticating with the given API token. It
// returns [ErrTokenChecksum] for a token whose checksum is wrong.
func New(token string, opts ...Option) (*Client, error) {
	if token == "" {
		return nil, errors.New("operator: empty API token")
	}
	if !strings.HasPrefix(token, TokenPrefix) {
		return nil, fmt.Errorf("operator: API token must start with %q", TokenPrefix)
	}
	if err := checkToken(token); err != nil {
		return nil, err
	}
	c := &Client{
		baseURL: DefaultBaseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.userAgent = strings.TrimSpace(c.userAgent + " operator-go/" + Version)
	if _, err := url.Parse(c.baseURL); err != nil {
		return nil, fmt.Errorf("operator: bad base URL %q: %w", c.baseURL, err)
	}
	return c, nil
}

// teamPlaceholder starts the path of every team-scoped route. Requests replace it with the
// client's team, so every team route names its team (API v1).
const teamPlaceholder = "{team}"

// ErrNoTeam is returned when the client has no team set and the credential can act on none, or
// on more than one, so there is no single team to pick. Set one with [WithTeam].
var ErrNoTeam = errors.New("operator: no single team for this credential; set one with WithTeam")

// TeamID returns the team the client acts on, asking the API once if [WithTeam] wasn't given.
func (c *Client) TeamID(ctx context.Context) (string, error) {
	c.teamMu.Lock()
	defer c.teamMu.Unlock()
	if c.teamID != "" {
		return c.teamID, nil
	}
	var page Page[Team]
	if err := c.do(ctx, http.MethodGet, "/v1/teams?limit=2", nil, &page); err != nil {
		return "", err
	}
	if len(page.Items) != 1 || page.Items[0] == nil {
		return "", ErrNoTeam
	}
	c.teamID = page.Items[0].ID
	return c.teamID, nil
}

// resolvePath fills in the team for a team-scoped path.
func (c *Client) resolvePath(ctx context.Context, path string) (string, error) {
	rest, ok := strings.CutPrefix(path, teamPlaceholder)
	if !ok {
		return path, nil
	}
	teamID, err := c.TeamID(ctx)
	if err != nil {
		return "", err
	}
	return "/v1/teams/" + url.PathEscape(teamID) + rest, nil
}

// DryRun reports whether the client is in dry-run mode.
func (c *Client) DryRun() bool { return c.dryRun }

// envelope is the standard API response wrapper.
type envelope struct {
	Data      json.RawMessage `json:"data,omitempty"`
	Message   string          `json:"message,omitempty"`
	Code      string          `json:"code,omitempty"`
	Status    int             `json:"status,omitempty"`
	RequestID string          `json:"requestId,omitempty"`
}

func (c *Client) mutating(method string) bool {
	return method != http.MethodGet
}

// do issues an HTTP request and decodes the envelope's data field into out
// (when out is non-nil). Mutating requests in dry-run mode short-circuit with
// a [*DryRunError].
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	return c.doAuth(ctx, method, path, body, out, "")
}

// doAuth is do with a bearer credential override. The webhook trigger route
// authenticates with an automation trigger secret rather than the member API
// token; every other route passes an empty override.
func (c *Client) doAuth(ctx context.Context, method, path string, body, out any, bearerOverride string) error {
	path, err := c.resolvePath(ctx, path)
	if err != nil {
		return err
	}
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
	bearer := c.token
	if bearerOverride != "" {
		bearer = bearerOverride
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == http.MethodPost {
		key, err := idempotencyKey(ctx)
		if err != nil {
			return err
		}
		req.Header.Set(IdempotencyHeader, key)
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
			Code:       env.Code,
			RetryAfter: retryAfter(res.Header.Get("Retry-After")),
			RequestID:  requestID(env, res.Header),
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

// pageQuery renders cursor/limit query parameters, omitting zero values.
// filterQuery is pageQuery with one filter, which is left out when empty.
func filterQuery(key, value, cursor string, limit int) string {
	v := url.Values{}
	if value != "" {
		v.Set(key, value)
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	if limit > 0 {
		v.Set("limit", fmt.Sprint(limit))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

func pageQuery(cursor string, limit int) string {
	return filterQuery("", "", cursor, limit)
}

// WalkLimit is the platform's largest page. Pass it to a List method inside
// [All] so a walk makes the fewest requests.
const WalkLimit = 100

// maxWalkPages bounds a walk against a list whose cursor never ends.
const maxWalkPages = 1000

// ErrEndlessList is returned by [All] when a list still has a next cursor
// after the most pages a walk reads.
var ErrEndlessList = errors.New("operator: list has no last page")

// All walks a list from its first page to its last and yields each item.
// fetch reads the page at a cursor, where the empty cursor is the first page:
//
//	for d, err := range operator.All(ctx, func(ctx context.Context, cursor string) (operator.Page[operator.Device], error) {
//		return client.ListDevices(ctx, cursor, operator.WalkLimit)
//	}) {
//
// The walk stops at the first error, which it yields with a nil item.
func All[T any](ctx context.Context, fetch func(ctx context.Context, cursor string) (Page[T], error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		cursor := ""
		for range maxWalkPages {
			page, err := fetch(ctx, cursor)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, item := range page.Items {
				if item != nil && !yield(item, nil) {
					return
				}
			}
			if page.NextCursor == nil || *page.NextCursor == "" {
				return
			}
			cursor = *page.NextCursor
		}
		yield(nil, ErrEndlessList)
	}
}

// allPages reads a cursor-paged list to its end and returns every item.
// path may already carry a query.
func allPages[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	fetch := func(ctx context.Context, cursor string) (Page[T], error) {
		var page Page[T]
		q := strings.TrimPrefix(pageQuery(cursor, WalkLimit), "?")
		err := c.do(ctx, http.MethodGet, path+sep+q, nil, &page)
		return page, err
	}
	var out []T
	for item, err := range All(ctx, fetch) {
		if err != nil {
			if errors.Is(err, ErrEndlessList) {
				return nil, fmt.Errorf("operator: listing %s: %w", path, err)
			}
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

// retryAfter parses a Retry-After header given in seconds; anything else is zero.
func retryAfter(header string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
