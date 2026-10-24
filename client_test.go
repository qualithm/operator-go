package operator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc adapts a function to an [http.RoundTripper] so tests can
// intercept requests without a live server.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTestClient(t *testing.T, fn roundTripFunc, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithHTTPClient(&http.Client{Transport: fn}), WithTeam("team_1")}
	c, err := New("qmt_sel.ver", append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewValidatesToken(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("expected error for empty token")
	}
	if _, err := New("nope_123"); err == nil {
		t.Fatal("expected error for missing prefix")
	}
	if _, err := New("qmt_sel.ver"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewChecksTokenChecksum(t *testing.T) {
	t.Parallel()
	selector := strings.Repeat("a", 24)
	verifier := strings.Repeat("0123456789abcdef", 4)
	legacy := "qmt_" + selector + "." + verifier
	issued := legacy + fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(legacy)))
	last := "0"
	if strings.HasSuffix(issued, "0") {
		last = "1"
	}

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{"issued with a checksum", issued, nil},
		{"issued before the checksum", legacy, nil},
		{"checksum character changed", issued[:len(issued)-1] + last, ErrTokenChecksum},
		{"verifier character changed", strings.Replace(issued, ".0", ".f", 1), ErrTokenChecksum},
		{"checksum-length verifier that is not hex", legacy + "zzzzzzzz", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New(%q) error = %v, want %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestGetDecodesData(t *testing.T) {
	var gotAuth, gotPath string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		gotAuth = req.Header.Get("Authorization")
		gotPath = req.URL.Path + "?" + req.URL.RawQuery
		return jsonResponse(200, `{"data":{"current":1,"items":[{"id":"dev_1","name":"a"}],"last":1}}`), nil
	})
	page, err := c.ListDevices(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if gotAuth != "Bearer qmt_sel.ver" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotPath != "/v1/teams/team_1/devices?limit=20&page=1" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "dev_1" {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestUserAgentHeader(t *testing.T) {
	var got string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get("User-Agent")
		return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	})
	if _, err := c.ListDevices(context.Background(), 1, 20); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if want := "operator-go/" + Version; got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
}

func TestWithUserAgentPrefix(t *testing.T) {
	var got string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get("User-Agent")
		return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	}, WithUserAgent("qualithm/9.9.9"))
	if _, err := c.ListDevices(context.Background(), 1, 20); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if want := "qualithm/9.9.9 operator-go/" + Version; got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
}

func TestErrorStatusReturnsClientError(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(404, `{"message":"Device not found"}`), nil
	})
	_, err := c.GetDevice(context.Background(), "dev_x")
	var ce *ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClientError, got %T: %v", err, err)
	}
	if ce.StatusCode != 404 || ce.Message != "Device not found" {
		t.Fatalf("unexpected ClientError: %+v", ce)
	}
}

func TestUnavailableCarriesCodeAndRetryAfter(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		res := jsonResponse(503, `{"message":"service temporarily unavailable","code":"backend_unavailable"}`)
		res.Header.Set("Retry-After", "5")
		return res, nil
	})
	_, err := c.GetDevice(context.Background(), "dev_x")
	var ce *ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClientError, got %T: %v", err, err)
	}
	if !ce.Unavailable() || ce.Code != "backend_unavailable" || ce.RetryAfter != 5*time.Second {
		t.Fatalf("unexpected ClientError: %+v", ce)
	}
}

func TestRetryAfterIgnoresNonSeconds(t *testing.T) {
	for _, header := range []string{"", "soon", "-1", "Wed, 21 Oct 2026 07:28:00 GMT"} {
		if got := retryAfter(header); got != 0 {
			t.Errorf("retryAfter(%q) = %v, want 0", header, got)
		}
	}
}

func TestDryRunShortCircuitsMutations(t *testing.T) {
	called := false
	var recorded Action
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(201, `{}`), nil
	}, WithDryRun(true), WithRecorder(func(a Action) { recorded = a }))

	_, err := c.CreateEnrollment(context.Background(), CreateEnrollmentInput{SpaceID: "spc_1"})
	var dre *DryRunError
	if !errors.As(err, &dre) {
		t.Fatalf("expected DryRunError, got %T: %v", err, err)
	}
	if called {
		t.Fatal("mutation should not hit the transport in dry-run")
	}
	if dre.Action.Method != http.MethodPost || dre.Action.Path != "/v1/teams/team_1/enrollments" {
		t.Fatalf("unexpected action: %+v", dre.Action)
	}
	if recorded.Path != "/v1/teams/team_1/enrollments" {
		t.Fatalf("recorder not invoked: %+v", recorded)
	}
}

func TestDryRunStillReads(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	}, WithDryRun(true))
	if _, err := c.ListDevices(context.Background(), 0, 0); err != nil {
		t.Fatalf("read in dry-run should succeed: %v", err)
	}
}

func TestMintCredentialSendsBody(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		_ = json.NewDecoder(req.Body).Decode(&body)
		return jsonResponse(201, `{"data":{"credential":{"id":"cred_1"},"secret":"qmd_abc"}}`), nil
	})
	res, err := c.MintCredential(context.Background(), "dev_1", MintCredentialInput{Label: "lab"})
	if err != nil {
		t.Fatalf("MintCredential: %v", err)
	}
	if res.Secret != "qmd_abc" || res.Credential.ID != "cred_1" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if body["label"] != "lab" {
		t.Fatalf("request body = %+v", body)
	}
}

func TestPageQuery(t *testing.T) {
	cases := map[string]string{
		pageQuery(0, 0):   "",
		pageQuery(2, 0):   "?page=2",
		pageQuery(0, 50):  "?limit=50",
		pageQuery(3, 100): "?limit=100&page=3",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("pageQuery = %q, want %q", got, want)
		}
	}
}

func TestTeamIDAsksOnceWhenUnset(t *testing.T) {
	var calls []string
	c, err := New("qmt_sel.ver", WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.URL.Path)
		if req.URL.Path == "/v1/teams" {
			return jsonResponse(200, `{"data":{"current":1,"items":[{"id":"team_9"}],"last":1}}`), nil
		}
		return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := c.ListDevices(context.Background(), 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"/v1/teams", "/v1/teams/team_9/devices", "/v1/teams/team_9/devices"}
	if strings.Join(calls, " ") != strings.Join(want, " ") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestTeamIDNeedsExactlyOneTeam(t *testing.T) {
	c, err := New("qmt_sel.ver", WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"current":1,"items":[{"id":"a"},{"id":"b"}],"last":1}}`), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListDevices(context.Background(), 0, 0); !errors.Is(err, ErrNoTeam) {
		t.Fatalf("err = %v, want ErrNoTeam", err)
	}
}
