package operator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
	page, err := c.ListAuthorities(context.Background(), 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/authorities" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if rec.query != "limit=50&page=2" {
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
	if rec.method != http.MethodPost || rec.path != "/authorities" {
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
	if rec.method != http.MethodDelete || rec.path != "/authorities/aut_9" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListCredentialsDecodesArray(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":[{"id":"cred_1","kind":"token"},{"id":"cred_2","kind":"cert"}]}`, &rec)
	creds, err := c.ListCredentials(context.Background(), "dev_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/devices/dev_1/credentials" {
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
	if rec.method != http.MethodPost || rec.path != "/devices/dev_1/credentials/cert" {
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
	if rec.path != "/devices/dev_1/credentials/cred_1/rotate" || rec.query != "revoke=true" {
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
	if rec.method != http.MethodDelete || rec.path != "/devices/dev_1/credentials/cred_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListEnrollments(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"enr_1","spaceId":"spc_1"}],"last":1}}`, &rec)
	page, err := c.ListEnrollments(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/enrollments" || rec.query != "" {
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
	if rec.method != http.MethodDelete || rec.path != "/enrollments/enr_9" {
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
	if rec.method != http.MethodPost || rec.path != "/devices" || rec.body["spaceId"] != "spc_1" {
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
	if rec.method != http.MethodPatch || rec.path != "/devices/dev_1" || rec.body["name"] != "gw" {
		t.Fatalf("request = %s %s body=%+v", rec.method, rec.path, rec.body)
	}
}

func TestDeleteDevice(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.DeleteDevice(context.Background(), "dev_1"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/devices/dev_1" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
}

func TestListSpaceDevices(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"dev_1"}],"last":1}}`, &rec)
	if _, err := c.ListSpaceDevices(context.Background(), "spc_1", 1, 100); err != nil {
		t.Fatal(err)
	}
	if rec.path != "/spaces/spc_1/devices" || rec.query != "limit=100&page=1" {
		t.Fatalf("request = %s?%s", rec.path, rec.query)
	}
}

func TestListAPITokens(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"data":{"current":1,"items":[{"id":"tok_1","name":"ci"}],"last":1}}`, &rec)
	page, err := c.ListAPITokens(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/api-tokens" {
		t.Fatalf("path = %q", rec.path)
	}
	if page.Items[0].Name != "ci" {
		t.Fatalf("page = %+v", page)
	}
}

func TestCreateAPIToken(t *testing.T) {
	var rec recorder
	c := recClient(t, 201, `{"data":{"token":{"id":"tok_1"},"secret":"qmt_new"}}`, &rec)
	got, err := c.CreateAPIToken(context.Background(), CreateAPITokenInput{Name: "ci", ExpiresInDays: 90})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/api-tokens" {
		t.Fatalf("request = %s %s", rec.method, rec.path)
	}
	if got.Secret != "qmt_new" || got.Token.ID != "tok_1" {
		t.Fatalf("result = %+v", got)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	var rec recorder
	c := recClient(t, 200, `{"message":"ok"}`, &rec)
	if err := c.RevokeAPIToken(context.Background(), "tok_9"); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/api-tokens/tok_9" {
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
	if _, err := c.ListDevices(context.Background(), 0, 0); err != nil {
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
	withMsg := &ClientError{Method: "GET", Path: "/devices", StatusCode: 500, Message: "boom"}
	if got := withMsg.Error(); got != "operator: GET /devices: HTTP 500: boom" {
		t.Fatalf("Error() = %q", got)
	}
	noMsg := &ClientError{Method: "DELETE", Path: "/devices/dev_1", StatusCode: 502}
	if got := noMsg.Error(); got != "operator: DELETE /devices/dev_1: HTTP 502" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestDryRunErrorMessageFormat(t *testing.T) {
	e := &DryRunError{Action: Action{Method: "POST", Path: "/devices"}}
	if got := e.Error(); got != "dry-run: would POST /devices" {
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
	_, err := c.ListDevices(context.Background(), 0, 0)
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
