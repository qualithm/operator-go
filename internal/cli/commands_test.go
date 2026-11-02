package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run is a convenience wrapper that executes Run with a background context.
func run(env Env, args ...string) int {
	return Run(context.Background(), env, args)
}

const tok = "qmt_x.y"

func TestDefaultEnvIsWired(t *testing.T) {
	e := DefaultEnv()
	if e.Stdin == nil || e.Stdout == nil || e.Stderr == nil || e.NewClient == nil {
		t.Fatal("DefaultEnv left a field nil")
	}
}

func TestTopLevelHelpAndUnknown(t *testing.T) {
	env, out, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "--help"); code != ExitOK {
		t.Fatalf("help exit = %d", code)
	}
	if !strings.Contains(out.String(), "operator CLI") {
		t.Fatalf("help stdout = %q", out.String())
	}
	env2, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "bogus"); code != ExitUsage {
		t.Fatalf("unknown exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown command") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestNoArgsIsUsage(t *testing.T) {
	env, _, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
}

// --- authorities -----------------------------------------------------------

func TestAuthorityListTable(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"aut_1","name":"ca","kind":"platform","status":"active","fingerprint":"ab"}],"last":1}}`), nil
	})
	if code := run(env, "authority", "list", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "aut_1") || !strings.Contains(got, "FINGERPRINT") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestAuthorityListJSON(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"aut_1"}],"last":1}}`), nil
	})
	if code := run(env, "authority", "list", "--token", tok, "--json"); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), `"id": "aut_1"`) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestAuthorityCreatePlatform(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/teams/team_1/authorities" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return jsonResp(201, `{"data":{"id":"aut_1","name":"ca"}}`), nil
	})
	if code := run(env, "authority", "create", "--name", "ca", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "registered") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestAuthorityCreateMissingName(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "authority", "create", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--name is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestAuthorityCreateByoNeedsCertFile(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	code := run(env, "authority", "create", "--name", "ca", "--kind", "byo", "--token", tok)
	if code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--cert-file is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestAuthorityCreateByoReadsCertFile(t *testing.T) {
	dir := t.TempDir()
	pem := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(pem, []byte("PEMDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	var sentPEM string
	env, _, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		b := make([]byte, req.ContentLength)
		_, _ = req.Body.Read(b)
		sentPEM = string(b)
		return jsonResp(201, `{"data":{"id":"aut_1","name":"ca"}}`), nil
	})
	if code := run(env, "authority", "create", "--name", "ca", "--kind", "byo", "--cert-file", pem, "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(sentPEM, "PEMDATA") {
		t.Fatalf("body = %q", sentPEM)
	}
}

func TestAuthorityRevokeAndMissingID(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env, "authority", "revoke", "aut_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "revoked") {
		t.Fatalf("stdout = %q", out.String())
	}
	env2, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "authority", "revoke", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-id exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "authority id is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestAuthorityUnknownVerbAndHelp(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "authority", "frob"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown authority verb") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	env2, out, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "authority", "help"); code != ExitOK {
		t.Fatalf("help exit = %d", code)
	}
	if !strings.Contains(out.String(), "usage: qualithm authority") {
		t.Fatalf("stdout = %q", out.String())
	}
	env3, _, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env3, "authority"); code != ExitUsage {
		t.Fatalf("no-verb exit = %d", code)
	}
}

// --- enrollments -----------------------------------------------------------

func TestEnrollmentListAndRevoke(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"enr_1","scopeId":"spc_1","status":"pending"}],"last":1}}`), nil
	})
	if code := run(env, "enrollment", "list", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "enr_1") {
		t.Fatalf("stdout = %q", out.String())
	}
	env2, out2, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env2, "enrollment", "revoke", "enr_1", "--token", tok); code != ExitOK {
		t.Fatalf("revoke exit = %d", code)
	}
	if !strings.Contains(out2.String(), "revoked") {
		t.Fatalf("stdout = %q", out2.String())
	}
}

func TestEnrollmentCreateMissingScopeAndRevokeMissingID(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "enrollment", "create", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--scope is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	env2, _, errBuf2 := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "enrollment", "revoke", "--token", tok); code != ExitUsage {
		t.Fatalf("revoke exit = %d", code)
	}
	if !strings.Contains(errBuf2.String(), "enrollment id is required") {
		t.Fatalf("stderr = %q", errBuf2.String())
	}
}

func TestEnrollmentCreateHuman(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(201, `{"data":{"enrollment":{"id":"enr_1","scopeId":"spc_1"},"code":"qmc_abc"}}`), nil
	})
	if code := run(env, "enrollment", "create", "--scope", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "qmc_abc") {
		t.Fatalf("stdout = %q", out.String())
	}
}

// --- credentials -----------------------------------------------------------

func TestCredentialListRequiresDevice(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "credential", "list", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--device is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestCredentialListTable(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"cred_1","kind":"token","status":"active"}],"last":1,"nextCursor":null}}`), nil
	})
	if code := run(env, "credential", "list", "--device", "dev_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "cred_1") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCredentialCert(t *testing.T) {
	dir := t.TempDir()
	csr := filepath.Join(dir, "d.csr")
	if err := os.WriteFile(csr, []byte("CSRDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/teams/team_1/devices/dev_1/credentials/cert" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		return jsonResp(201, `{"data":{"credential":{"id":"cred_1","deviceId":"dev_1","certFingerprint":"fp"},"certificatePem":"LEAF\n","caCertificatePem":"CA\n"}}`), nil
	})
	if code := run(env, "credential", "cert", "--device", "dev_1", "--csr-file", csr, "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "LEAF") || !strings.Contains(got, "CA") || !strings.Contains(got, "fp") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestCredentialCertMissingFlags(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "credential", "cert", "--device", "dev_1", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--device and --csr-file are required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestCredentialCertUnreadableCSR(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	code := run(env, "credential", "cert", "--device", "dev_1", "--csr-file", "/no/such/file.csr", "--token", tok)
	if code != ExitError {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "read csr") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestCredentialRotateWithRevoke(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.RawQuery != "revoke=true" {
			t.Fatalf("query = %q", req.URL.RawQuery)
		}
		return jsonResp(200, `{"data":{"credential":{"id":"cred_2","rotatedFrom":"cred_1"},"secret":"qmd_x"}}`), nil
	})
	code := run(env, "credential", "rotate", "--device", "dev_1", "--credential", "cred_1", "--revoke", "--token", tok)
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "cred_1 -> cred_2") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCredentialRotateMissingFlags(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "credential", "rotate", "--device", "dev_1", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--device and --credential are required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestCredentialRevoke(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	code := run(env, "credential", "revoke", "--device", "dev_1", "--credential", "cred_1", "--token", tok)
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "cred_1 revoked") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCredentialUnknownVerb(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "credential", "frob"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown credential verb") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

// --- devices ---------------------------------------------------------------

func TestDeviceListInScope(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/teams/team_1/devices" || req.URL.Query().Get("scopeId") != "spc_1" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"dev_1","scopeId":"spc_1"}],"last":1}}`), nil
	})
	if code := run(env, "device", "list", "--scope", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "dev_1") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDeviceListCursor(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("cursor") != "c1" || req.URL.Query().Has("page") {
			t.Fatalf("query = %s", req.URL.RawQuery)
		}
		return jsonResp(200, `{"data":{"items":[{"id":"dev_2"}],"nextCursor":"c2"}}`), nil
	})
	if code := run(env, "device", "list", "--cursor", "c1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "more: --cursor c2") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDeviceGetTable(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"id":"dev_1","scopeId":"spc_1","name":"gw"}}`), nil
	})
	if code := run(env, "device", "get", "dev_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "gw") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDeviceGetMissingID(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "device", "get", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "device id is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestDeviceCreateAndMissingScope(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(201, `{"data":{"id":"dev_1","scopeId":"spc_1","name":""}}`), nil
	})
	if code := run(env, "device", "create", "--scope", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "created in scope spc_1") {
		t.Fatalf("stdout = %q", out.String())
	}
	env2, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "device", "create", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-scope exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--scope is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestDeviceUpdate(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("method = %s", req.Method)
		}
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env, "device", "update", "dev_1", "--name", "gw2", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "dev_1 updated") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDeviceUpdateRequiresAField(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "device", "update", "dev_1", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "set --name and/or --scope") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestDeviceDelete(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env, "device", "delete", "dev_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "dev_1 deleted") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDeviceUnknownVerbAndNoVerb(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "device", "frob"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown device verb") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	env2, _, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "device"); code != ExitUsage {
		t.Fatalf("no-verb exit = %d", code)
	}
}

// --- scopes ----------------------------------------------------------------

func TestScopeList(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/teams/team_1/scopes" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"spc_1","name":"edge","zone":"de-fra-a","deviceTotal":3}],"last":1}}`), nil
	})
	if code := run(env, "scope", "list", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "spc_1") || !strings.Contains(out.String(), "de-fra-a") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeListInZone(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/teams/team_1/scopes" || req.URL.Query().Get("zone") != "sg-sin-a" {
			t.Fatalf("request = %s?%s", req.URL.Path, req.URL.RawQuery)
		}
		return jsonResp(200, `{"data":{"current":1,"items":[null,{"id":"spc_2","name":"bench","zone":"sg-sin-a","parentId":"spc_1"}],"last":1}}`), nil
	})
	if code := run(env, "scope", "list", "--zone", "sg-sin-a", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "spc_2") || !strings.Contains(out.String(), "spc_1") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeGetTable(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1/teams/team_1/scopes/spc_1" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		return jsonResp(200, `{"data":{"id":"spc_1","name":"edge","zone":"sg-sin-a","deviceTotal":0}}`), nil
	})
	if code := run(env, "scope", "get", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "sg-sin-a") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeGetMissingID(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "scope", "get", "--token", tok); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "scope id is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestScopeCreateTopLevel(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/teams/team_1/scopes" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		if !strings.Contains(string(body), `"name":"hall"`) || !strings.Contains(string(body), `"zone":"de-fra-a"`) || strings.Contains(string(body), "parentId") {
			t.Fatalf("body = %s", body)
		}
		return jsonResp(201, `{"data":{"id":"spc_1","name":"hall","zone":"de-fra-a","parentId":null}}`), nil
	})
	if code := run(env, "scope", "create", "--name", "hall", "--zone", "de-fra-a", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "scope spc_1 (hall) created in zone de-fra-a") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeCreateChild(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if !strings.Contains(string(body), `"parentId":"spc_1"`) || strings.Contains(string(body), "zone") {
			t.Fatalf("body = %s", body)
		}
		return jsonResp(201, `{"data":{"id":"spc_2","name":"bench","zone":"de-fra-a","parentId":"spc_1"}}`), nil
	})
	if code := run(env, "scope", "create", "--name", "bench", "--parent", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "scope spc_2 (bench) created in zone de-fra-a") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeCreateMissingFlags(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "scope", "create", "--zone", "de-fra-a", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-name exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--name is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	env2, _, errBuf2 := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "scope", "create", "--name", "hall", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-zone exit = %d", code)
	}
	if !strings.Contains(errBuf2.String(), "set --zone for a top-level scope or --parent for a child") {
		t.Fatalf("stderr = %q", errBuf2.String())
	}
}

func TestScopeUpdateAndMissingName(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch || req.URL.Path != "/v1/teams/team_1/scopes/spc_1" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env, "scope", "update", "spc_1", "--name", "hall", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "renamed to hall") {
		t.Fatalf("stdout = %q", out.String())
	}
	env2, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "scope", "update", "spc_1", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-name exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "--name is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestScopeDelete(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			t.Fatalf("method = %s", req.Method)
		}
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env, "scope", "delete", "spc_1", "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "spc_1 deleted") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeRejectsBadInput(t *testing.T) {
	t.Setenv("QUALITHM_API_TOKEN", "")
	noCall := func(*http.Request) (*http.Response, error) {
		t.Fatal("no request expected")
		return nil, nil
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"list bad flag", []string{"scope", "list", "--frob"}, "frob"},
		{"list no token", []string{"scope", "list"}, "API token is required"},
		{"get bad flag", []string{"scope", "get", "--frob"}, "frob"},
		{"get no token", []string{"scope", "get", "spc_1"}, "API token is required"},
		{"create bad flag", []string{"scope", "create", "--frob"}, "frob"},
		{"create no token", []string{"scope", "create", "--name", "hall", "--zone", "de-fra-a"}, "API token is required"},
		{"update bad flag", []string{"scope", "update", "--frob"}, "frob"},
		{"update missing id", []string{"scope", "update", "--name", "hall", "--token", tok}, "scope id is required"},
		{"update no token", []string{"scope", "update", "spc_1", "--name", "hall"}, "API token is required"},
		{"delete bad flag", []string{"scope", "delete", "--frob"}, "frob"},
		{"delete missing id", []string{"scope", "delete", "--token", tok}, "scope id is required"},
		{"delete no token", []string{"scope", "delete", "spc_1"}, "API token is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, _, errBuf := testEnv(noCall)
			if code := run(env, tc.args...); code != ExitUsage {
				t.Fatalf("exit = %d", code)
			}
			if !strings.Contains(errBuf.String(), tc.want) {
				t.Fatalf("stderr = %q", errBuf.String())
			}
		})
	}
}

func TestScopeHelp(t *testing.T) {
	env, out, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "scope", "help"); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "usage: qualithm scope") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestScopeUnknownVerbAndNoVerb(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "scope", "frob"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown scope verb") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	env2, _, _ := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "scope"); code != ExitUsage {
		t.Fatalf("no-verb exit = %d", code)
	}
}

// --- tokens ----------------------------------------------------------------

func TestTokenListRevokeAndCreateRefused(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"tok_1","name":"ci","status":"active","role":"viewer"},{"id":"tok_2","name":"old","status":"active","role":null}],"last":1}}`), nil
	})
	if code := run(env, "token", "list", "--token", tok); code != ExitOK {
		t.Fatalf("list exit = %d", code)
	}
	if !strings.Contains(out.String(), "viewer") || !strings.Contains(out.String(), "your role") {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(out.String(), "tok_1") {
		t.Fatalf("stdout = %q", out.String())
	}
	env2, _, errBuf2 := testEnv(func(*http.Request) (*http.Response, error) {
		t.Fatal("create must not call the API")
		return nil, nil
	})
	if code := run(env2, "token", "create", "--name", "ci", "--token", tok); code != ExitUsage {
		t.Fatalf("create exit = %d", code)
	}
	if !strings.Contains(errBuf2.String(), "in the app") {
		t.Fatalf("stderr = %q", errBuf2.String())
	}
	env3, out3, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"message":"ok"}`), nil
	})
	if code := run(env3, "token", "revoke", "tok_1", "--token", tok); code != ExitOK {
		t.Fatalf("revoke exit = %d", code)
	}
	if !strings.Contains(out3.String(), "tok_1 revoked") {
		t.Fatalf("stdout = %q", out3.String())
	}
	env4, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env4, "token", "revoke", "--token", tok); code != ExitUsage {
		t.Fatalf("missing-id exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "token id is required") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestTokenUnknownVerb(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "token", "frob"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown token verb") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

// --- error mapping & dry-run ----------------------------------------------

func TestStatusToExitCodeMapping(t *testing.T) {
	cases := map[int]int{
		401: ExitAuth,
		403: ExitAuth,
		404: ExitNotFound,
		409: ExitConflict,
		429: ExitRateLimited,
		500: ExitAPI,
		503: ExitUnavailable,
	}
	for status, want := range cases {
		env, _, _ := testEnv(func(req *http.Request) (*http.Response, error) {
			return jsonResp(status, `{"message":"nope"}`), nil
		})
		if code := run(env, "device", "list", "--token", tok); code != want {
			t.Errorf("status %d -> exit %d, want %d", status, code, want)
		}
	}
}

func TestBadTokenPrefixIsUsage(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "device", "list", "--token", "bad"); code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "must start with") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}

func TestDryRunJSONOutput(t *testing.T) {
	env, out, _ := testEnv(func(*http.Request) (*http.Response, error) {
		t.Fatal("dry-run must not hit network")
		return nil, nil
	})
	code := run(env, "device", "delete", "dev_1", "--token", tok, "--dry-run", "--json")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, `"dryRun": true`) || !strings.Contains(got, `"method": "DELETE"`) {
		t.Fatalf("stdout = %q", got)
	}
}

func TestDryRunHumanShowsBody(t *testing.T) {
	env, out, _ := testEnv(func(*http.Request) (*http.Response, error) {
		t.Fatal("dry-run must not hit network")
		return nil, nil
	})
	code := run(env, "device", "update", "dev_1", "--name", "gw", "--token", tok, "--dry-run")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "would PATCH /v1/teams/team_1/devices/dev_1") || !strings.Contains(got, "body:") {
		t.Fatalf("stdout = %q", got)
	}
}

// --- apply -----------------------------------------------------------------

func writeManifest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestApplyCreatesMissingDevice(t *testing.T) {
	var posted, patched bool
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet:
			return jsonResp(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/teams/team_1/devices":
			posted = true
			return jsonResp(201, `{"data":{"id":"dev_new","scopeId":"spc_1"}}`), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/teams/team_1/devices/dev_new":
			patched = true
			return jsonResp(200, `{"message":"ok"}`), nil
		}
		t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		return nil, nil
	})
	path := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: gw-1\n")
	if code := run(env, "apply", path, "--token", tok); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !posted || !patched {
		t.Fatalf("posted=%v patched=%v", posted, patched)
	}
	if !strings.Contains(out.String(), "create") || !strings.Contains(out.String(), "dev_new") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestApplyExistingDeviceIsNoOp(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("existing device must not mutate: %s %s", req.Method, req.URL.Path)
		}
		return jsonResp(200, `{"data":{"current":1,"items":[{"id":"dev_1","name":"gw-1"}],"last":1}}`), nil
	})
	path := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: gw-1\n")
	if code := run(env, "apply", path, "--token", tok, "--json"); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), `"exists"`) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestApplyReadsEveryScopePage(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("a device on a later page must not be created: %s %s", req.Method, req.URL.Path)
		}
		if req.URL.Query().Get("cursor") == "" {
			return jsonResp(200, `{"data":{"items":[{"id":"dev_1","name":"gw-1"}],"nextCursor":"c2"}}`), nil
		}
		return jsonResp(200, `{"data":{"items":[{"id":"dev_2","name":"gw-2"}],"nextCursor":null}}`), nil
	})
	path := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: gw-2\n")
	if code := run(env, "apply", path, "--token", tok, "--json"); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), `"exists"`) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestApplyMissingFileAndBadYAML(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env, "apply", "/no/such/manifest.yaml", "--token", tok); code != ExitError {
		t.Fatalf("missing exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "read manifest") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
	bad := writeManifest(t, "scopes: [oops\n")
	env2, _, errBuf2 := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env2, "apply", bad, "--token", tok); code != ExitUsage {
		t.Fatalf("bad-yaml exit = %d", code)
	}
	if !strings.Contains(errBuf2.String(), "parse manifest") {
		t.Fatalf("stderr = %q", errBuf2.String())
	}
}

func TestApplyEmptyAndMissingIDsAndNoPath(t *testing.T) {
	env, _, errBuf := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	empty := writeManifest(t, "scopes: []\n")
	if code := run(env, "apply", empty, "--token", tok); code != ExitUsage {
		t.Fatalf("empty exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "no scopes") {
		t.Fatalf("stderr = %q", errBuf.String())
	}

	env2, _, errBuf2 := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	noID := writeManifest(t, "scopes:\n  - devices:\n      - name: gw\n")
	if code := run(env2, "apply", noID, "--token", tok); code != ExitUsage {
		t.Fatalf("no-id exit = %d", code)
	}
	if !strings.Contains(errBuf2.String(), "missing an id") {
		t.Fatalf("stderr = %q", errBuf2.String())
	}

	env3, _, errBuf3 := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	})
	noName := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: \"\"\n")
	if code := run(env3, "apply", noName, "--token", tok); code != ExitUsage {
		t.Fatalf("no-name exit = %d", code)
	}
	if !strings.Contains(errBuf3.String(), "missing a name") {
		t.Fatalf("stderr = %q", errBuf3.String())
	}

	env4, _, errBuf4 := testEnv(func(*http.Request) (*http.Response, error) { return nil, nil })
	if code := run(env4, "apply", "--token", tok); code != ExitUsage {
		t.Fatalf("no-path exit = %d", code)
	}
	if !strings.Contains(errBuf4.String(), "manifest path is required") {
		t.Fatalf("stderr = %q", errBuf4.String())
	}
}

func TestApplyDryRunWouldCreate(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("dry-run apply must not mutate: %s", req.Method)
		}
		return jsonResp(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
	})
	path := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: gw-1\n")
	if code := run(env, "apply", path, "--token", tok, "--dry-run"); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "would-create") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestApplyListErrorPropagates(t *testing.T) {
	env, _, errBuf := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(403, `{"message":"forbidden"}`), nil
	})
	path := writeManifest(t, "scopes:\n  - id: spc_1\n    devices:\n      - name: gw-1\n")
	if code := run(env, "apply", path, "--token", tok); code != ExitAuth {
		t.Fatalf("exit = %d, want %d", code, ExitAuth)
	}
	if !strings.Contains(errBuf.String(), "forbidden") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}
