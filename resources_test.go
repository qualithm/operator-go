package operator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// recorder captures the most recent request a method issued.
type recorder struct {
	method string
	path   string
	query  string
	body   map[string]any
}

// recClient returns a client whose transport records the request and replies
// with the given status and body.
func recClient(t *testing.T, status int, respBody string, rec *recorder, opts ...Option) *Client {
	t.Helper()
	fn := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rec.method = req.Method
		rec.path = req.URL.Path
		rec.query = req.URL.RawQuery
		if req.Body != nil {
			raw, _ := io.ReadAll(req.Body)
			if len(raw) > 0 {
				rec.body = map[string]any{}
				_ = json.Unmarshal(raw, &rec.body)
			}
		}
		return jsonResponse(status, respBody), nil
	})
	return newTestClient(t, fn, opts...)
}

func TestListAuthorities(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"aut_1","name":"ca"}],"last":1}}`, &rec)
	page, err := c.ListAuthorities(context.Background(), "c2", 50)
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/v1/teams/team_1/authorities" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if rec.query != "cursor=c2&limit=50" {
		t.Fatalf("query = %q", rec.query)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "aut_1" {
		t.Fatalf("page = %+v", page)
	}
}

func TestCreateAuthority(t *testing.T) {
	var rec recorder
	c := recClient(t, 201, `{"data":{"id":"aut_1","name":"ca","kind":"byo"}}`, &rec)
	got, err := c.CreateAuthority(context.Background(), CreateAuthorityInput{Name: "ca", Kind: "byo", CertificatePEM: "PEM"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/teams/team_1/authorities" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if rec.body["name"] != "ca" || rec.body["kind"] != "byo" || rec.body["certificatePem"] != "PEM" {
		t.Fatalf("body = %+v", rec.body)
	}
	if got.ID != "aut_1" {
		t.Fatalf("authority = %+v", got)
	}
}

func TestRevokeAuthority(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeAuthority(context.Background(), "aut_9"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/authorities/aut_9" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListCredentialsDecodesPage(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"cred_1","kind":"token"},{"id":"cred_2","kind":"cert"}],"last":1,"nextCursor":null}}`, &rec)
	creds, err := c.ListCredentials(context.Background(), "dev_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/devices/dev_1/credentials" {
		t.Fatalf("path = %q", rec.path)
	}
	if len(creds) != 2 || creds[1].Kind != "cert" {
		t.Fatalf("creds = %+v", creds)
	}
}

func TestIssueCert(t *testing.T) {
	var rec recorder
	c := recClient(t, 201, `{"data":{"credential":{"id":"cred_1"},"certificatePem":"LEAF","caCertificatePem":"CA"}}`, &rec)
	got, err := c.IssueCert(context.Background(), "dev_1", IssueCertInput{CSRPEM: "CSR", ExpiresInDays: 90})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/teams/team_1/devices/dev_1/credentials/cert" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if rec.body["csrPem"] != "CSR" {
		t.Fatalf("body = %+v", rec.body)
	}
	if got.CertificatePEM != "LEAF" || got.CACertificatePEM != "CA" {
		t.Fatalf("cert = %+v", got)
	}
}

func TestRotateCredentialRevokeQuery(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"credential":{"id":"cred_2","rotatedFrom":"cred_1"},"secret":"qmd_x"}}`, &rec)
	got, err := c.RotateCredential(context.Background(), "dev_1", "cred_1", MintCredentialInput{Label: "next"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/devices/dev_1/credentials/cred_1/rotate" || rec.query != "revoke=true" {
		t.Fatalf("request = %s?%s", rec.path, rec.query)
	}
	if got.Credential.RotatedFrom != "cred_1" || got.Secret != "qmd_x" {
		t.Fatalf("result = %+v", got)
	}
}

func TestRotateCredentialNoRevoke(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"credential":{"id":"cred_2"},"secret":"qmd_x"}}`, &rec)
	if _, err := c.RotateCredential(context.Background(), "dev_1", "cred_1", MintCredentialInput{}, false); err != nil {
		t.Fatal(err)
	}
	if rec.query != "" {
		t.Fatalf("query = %q, want empty", rec.query)
	}
}

func TestRevokeCredential(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeCredential(context.Background(), "dev_1", "cred_1"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/devices/dev_1/credentials/cred_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListEnrollments(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"enr_1","spaceId":"spc_1"}],"last":1}}`, &rec)
	page, err := c.ListEnrollments(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/enrollments" || rec.query != "" {
		t.Fatalf("request = %s?%s", rec.path, rec.query)
	}
	if page.Items[0].ID != "enr_1" {
		t.Fatalf("page = %+v", page)
	}
}

func TestRevokeEnrollment(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeEnrollment(context.Background(), "enr_9"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/enrollments/enr_9" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestCreateDevice(t *testing.T) {
	var rec recorder
	c := recClient(t, 201, `{"data":{"id":"dev_1","spaceId":"spc_1"}}`, &rec)
	got, err := c.CreateDevice(context.Background(), CreateDeviceInput{SpaceID: "spc_1"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/teams/team_1/devices" || rec.body["spaceId"] != "spc_1" {
		t.Fatalf("request = %s %s body=%+v", rec.method, rec.path, rec.body)
	}
	if got.ID != "dev_1" {
		t.Fatalf("device = %+v", got)
	}
}

func TestUpdateDevice(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateDevice(context.Background(), "dev_1", UpdateDeviceInput{Name: "gw"}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.path != "/v1/teams/team_1/devices/dev_1" || rec.body["name"] != "gw" {
		t.Fatalf("request = %s %s body=%+v", rec.method, rec.path, rec.body)
	}
}

func TestDeleteDevice(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteDevice(context.Background(), "dev_1"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/devices/dev_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListSpaceDevices(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"dev_1"}],"last":1}}`, &rec)
	if _, err := c.ListSpaceDevices(context.Background(), "spc_1", "", 100); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/devices" || rec.query != "limit=100&spaceId=spc_1" {
		t.Fatalf("request = %s?%s", rec.path, rec.query)
	}
}

func TestListSpaces(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"spc_1","zone":"de-fra-a","deviceTotal":2}],"last":1}}`, &rec)
	page, err := c.ListSpaces(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/spaces" || rec.query != "" {
		t.Fatalf("request = %s?%s", rec.path, rec.query)
	}
	if page.Items[0].ID != "spc_1" || page.Items[0].Zone != "de-fra-a" || page.Items[0].DeviceTotal != 2 {
		t.Fatalf("page = %+v", page)
	}
}

func TestGetSpace(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"id":"spc_1","name":"edge","zone":"sg-sin-a"}}`, &rec)
	got, err := c.GetSpace(context.Background(), "spc_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/spaces/spc_1" {
		t.Fatalf("path = %q", rec.path)
	}
	if got.Name != "edge" || got.Zone != "sg-sin-a" {
		t.Fatalf("space = %+v", got)
	}
}

func TestCreateSpace(t *testing.T) {
	var rec recorder
	c := recClient(t, 201, `{"data":{"id":"spc_1","zone":"de-fra-a"}}`, &rec)
	got, err := c.CreateSpace(context.Background(), CreateSpaceInput{Zone: "de-fra-a"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/teams/team_1/spaces" || rec.body["zone"] != "de-fra-a" {
		t.Fatalf("request = %s %s body=%+v", rec.method, rec.path, rec.body)
	}
	if got.ID != "spc_1" {
		t.Fatalf("space = %+v", got)
	}
}

func TestUpdateSpace(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.UpdateSpace(context.Background(), "spc_1", "edge"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.path != "/v1/teams/team_1/spaces/spc_1" || rec.body["name"] != "edge" {
		t.Fatalf("request = %s %s body=%+v", rec.method, rec.path, rec.body)
	}
}

func TestDeleteSpace(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteSpace(context.Background(), "spc_1"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/spaces/spc_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListAPITokens(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"tok_1","name":"ci"}],"last":1}}`, &rec)
	page, err := c.ListAPITokens(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/teams/team_1/api-tokens" {
		t.Fatalf("path = %q", rec.path)
	}
	if page.Items[0].Name != "ci" {
		t.Fatalf("page = %+v", page)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeAPIToken(context.Background(), "tok_9"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v1/teams/team_1/api-tokens/tok_9" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestWithBaseURLRoutesRequest(t *testing.T) {
	var gotHost string
	fn := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotHost = req.URL.Host
		return jsonResponse(200, `{"data":{"items":[],"current":0,"last":0}}`), nil
	})
	c := newTestClient(t, fn, WithBaseURL("http://example.test:9000/"))
	if _, err := c.ListDevices(context.Background(), "", 0); err != nil {
		t.Fatal(err)
	}
	if gotHost != "example.test:9000" {
		t.Fatalf("host = %q", gotHost)
	}
}

func TestDryRunGetter(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{}`, &rec, WithDryRun(true))
	if !c.DryRun() {
		t.Fatal("DryRun() = false, want true")
	}
	plain := recClient(t, 200, `{}`, &rec)
	if plain.DryRun() {
		t.Fatal("DryRun() = true, want false")
	}
}

func TestClientErrorMessageFormat(t *testing.T) {
	withMsg := &ClientError{Method: "GET", Path: "/v1/teams/team_1/devices", StatusCode: 500, Message: "boom"}
	if got := withMsg.Error(); got != "operator: GET /v1/teams/team_1/devices: HTTP 500: boom" {
		t.Fatalf("Error() = %q", got)
	}
	noMsg := &ClientError{Method: "DELETE", Path: "/v1/teams/team_1/devices/dev_1", StatusCode: 502}
	if got := noMsg.Error(); got != "operator: DELETE /v1/teams/team_1/devices/dev_1: HTTP 502" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestDryRunErrorMessageFormat(t *testing.T) {
	e := &DryRunError{Action: Action{Method: "POST", Path: "/v1/teams/team_1/devices"}}
	if got := e.Error(); got != "dry-run: would POST /v1/teams/team_1/devices" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestErrorWithoutMessageStillMapsStatus(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(409, `{}`), nil
	})
	_, err := c.CreateDevice(context.Background(), CreateDeviceInput{SpaceID: "spc_1"})
	var ce *ClientError
	if !errors.As(err, &ce) || ce.StatusCode != 409 {
		t.Fatalf("err = %v", err)
	}
}

func TestInvalidDataFieldIsDecodeError(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		// data is a string but ListDevices decodes into a Page object.
		return jsonResponse(200, `{"data":"not-an-object"}`), nil
	})
	_, err := c.ListDevices(context.Background(), "", 0)
	if err == nil {
		t.Fatal("expected decode error for malformed data field")
	}
}

func TestNonJSONErrorBodyFallsBackToRawMessage(t *testing.T) {
	c := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(502, "  upstream unavailable  "), nil
	})
	_, err := c.GetDevice(context.Background(), "dev_1")
	var ce *ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %T %v", err, err)
	}
	if ce.StatusCode != 502 || ce.Message != "upstream unavailable" {
		t.Fatalf("client error = %+v", ce)
	}
}

func TestClientErrorRequestID(t *testing.T) {
	fromBody := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		res := jsonResponse(500, `{"message":"boom","requestId":"req_body"}`)
		res.Header.Set("Request-Id", "req_header")
		return res, nil
	})
	_, err := fromBody.GetDevice(context.Background(), "dev_1")
	var ce *ClientError
	if !errors.As(err, &ce) || ce.RequestID != "req_body" {
		t.Fatalf("body source: err = %v", err)
	}
	if !strings.HasSuffix(ce.Error(), "HTTP 500: boom (request req_body)") {
		t.Fatalf("Error() = %q", ce.Error())
	}

	fromHeader := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		res := jsonResponse(502, "upstream unavailable")
		res.Header.Set("Request-Id", "req_header")
		return res, nil
	})
	_, err = fromHeader.GetDevice(context.Background(), "dev_1")
	if !errors.As(err, &ce) || ce.RequestID != "req_header" {
		t.Fatalf("header source: err = %v", err)
	}
}

func TestListCredentialsReadsEveryPage(t *testing.T) {
	var queries []string
	fn := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		queries = append(queries, req.URL.RawQuery)
		if req.URL.Query().Get("cursor") == "" {
			return jsonResponse(200, `{"data":{"current":1,"items":[{"id":"cred_1"},null],"last":2,"nextCursor":"c2"}}`), nil
		}
		return jsonResponse(200, `{"data":{"current":2,"items":[{"id":"cred_2"}],"last":2,"nextCursor":null}}`), nil
	})
	creds, err := newTestClient(t, fn).ListCredentials(context.Background(), "dev_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 || creds[0].ID != "cred_1" || creds[1].ID != "cred_2" {
		t.Fatalf("creds = %+v", creds)
	}
	if len(queries) != 2 || queries[0] != "limit=100" || queries[1] != "cursor=c2&limit=100" {
		t.Fatalf("queries = %q", queries)
	}
}

func TestAllPagesStops(t *testing.T) {
	t.Run("on an error", func(t *testing.T) {
		c := recClient(t, 500, `{"message":"boom"}`, &recorder{})
		if _, err := c.GetTelemetry(context.Background(), GetTelemetryInput{DeviceID: "dev_1", Metric: "temp"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("on a cursor that never ends", func(t *testing.T) {
		var calls int
		fn := roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return jsonResponse(200, `{"data":{"current":1,"items":[],"last":1,"nextCursor":"again"}}`), nil
		})
		_, err := newTestClient(t, fn).GetTeamDeviceState(context.Background(), "team_1")
		if err == nil || !strings.Contains(err.Error(), "no last page") || calls != maxWalkPages {
			t.Fatalf("err = %v after %d calls", err, calls)
		}
	})
}

func TestAllWalksEveryPage(t *testing.T) {
	next := func(s string) *string { return &s }
	pages := map[string]Page[Device]{
		"":   {Items: []*Device{{ID: "dev_1"}, nil}, NextCursor: next("c2")},
		"c2": {Items: []*Device{{ID: "dev_2"}}, NextCursor: next("c3")},
		"c3": {Items: []*Device{{ID: "dev_3"}}},
	}
	var cursors []string
	fetch := func(_ context.Context, cursor string) (Page[Device], error) {
		cursors = append(cursors, cursor)
		page, ok := pages[cursor]
		if !ok {
			return Page[Device]{}, fmt.Errorf("unknown cursor %q", cursor)
		}
		return page, nil
	}

	t.Run("to the last page", func(t *testing.T) {
		cursors = nil
		var ids []string
		for d, err := range All(context.Background(), fetch) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, d.ID)
		}
		if strings.Join(ids, ",") != "dev_1,dev_2,dev_3" || strings.Join(cursors, ",") != ",c2,c3" {
			t.Fatalf("ids = %q cursors = %q", ids, cursors)
		}
	})
	t.Run("until the caller stops", func(t *testing.T) {
		cursors = nil
		for range All(context.Background(), fetch) {
			break
		}
		if len(cursors) != 1 {
			t.Fatalf("cursors = %q", cursors)
		}
	})
	t.Run("at the first error", func(t *testing.T) {
		boom := errors.New("boom")
		var errs []error
		for d, err := range All(context.Background(), func(context.Context, string) (Page[Device], error) {
			return Page[Device]{}, boom
		}) {
			if d != nil {
				t.Fatalf("item with an error: %+v", d)
			}
			errs = append(errs, err)
		}
		if len(errs) != 1 || !errors.Is(errs[0], boom) {
			t.Fatalf("errs = %v", errs)
		}
	})
	t.Run("on a cursor that never ends", func(t *testing.T) {
		var last error
		for _, err := range All(context.Background(), func(context.Context, string) (Page[Device], error) {
			return Page[Device]{NextCursor: next("again")}, nil
		}) {
			last = err
		}
		if !errors.Is(last, ErrEndlessList) {
			t.Fatalf("err = %v", last)
		}
	})
}
