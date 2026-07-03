package operator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc adapts a function to an [http.RoundTripper] so tests can
// intercept requests without a live server.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTestClient(t *testing.T, fn roundTripFunc, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithHTTPClient(&http.Client{Transport: fn})}
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
	if gotPath != "/devices?limit=20&page=1" {
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

func TestWithUserAgentOverride(t *testing.T) {
	var got string
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get("User-Agent")
		return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	}, WithUserAgent("qualithm/9.9.9"))
	if _, err := c.ListDevices(context.Background(), 1, 20); err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if got != "qualithm/9.9.9" {
		t.Fatalf("User-Agent = %q, want %q", got, "qualithm/9.9.9")
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
	if dre.Action.Method != http.MethodPost || dre.Action.Path != "/enrollments" {
		t.Fatalf("unexpected action: %+v", dre.Action)
	}
	if recorded.Path != "/enrollments" {
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
