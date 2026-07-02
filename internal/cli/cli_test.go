package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	operator "github.com/qualithm/operator-go"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// testEnv wires an Env whose client is backed by fn, capturing stdout/stderr.
func testEnv(fn roundTripFunc) (Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	env := Env{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &errBuf,
		NewClient: func(token string, opts ...operator.Option) (*operator.Client, error) {
			opts = append(opts, operator.WithHTTPClient(&http.Client{Transport: fn}))
			return operator.New(token, opts...)
		},
	}
	return env, &out, &errBuf
}

func TestCredentialMintJSON(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/devices/dev_1/credentials" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		return jsonResp(201, `{"data":{"credential":{"id":"cred_1","deviceId":"dev_1"},"secret":"qmd_abc"}}`), nil
	})
	code := Run(context.Background(), env, []string{"credential", "mint", "--device", "dev_1", "--token", "qmt_x.y", "--json"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), `"secret": "qmd_abc"`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestEnrollmentCreateDryRun(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		t.Fatal("dry-run must not hit the network")
		return nil, nil
	})
	code := Run(context.Background(), env, []string{"enrollment", "create", "--space", "spc_1", "--token", "qmt_x.y", "--dry-run"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "dry-run: would POST /enrollments") {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestNotFoundMapsToExitCode(t *testing.T) {
	env, _, errBuf := testEnv(func(req *http.Request) (*http.Response, error) {
		return jsonResp(404, `{"message":"Device not found"}`), nil
	})
	code := Run(context.Background(), env, []string{"device", "get", "dev_x", "--token", "qmt_x.y"})
	if code != ExitNotFound {
		t.Fatalf("exit = %d, want %d", code, ExitNotFound)
	}
	if !strings.Contains(errBuf.String(), "Device not found") {
		t.Fatalf("stderr = %s", errBuf.String())
	}
}

func TestMissingTokenIsUsageError(t *testing.T) {
	env, _, _ := testEnv(func(req *http.Request) (*http.Response, error) { return nil, nil })
	code := Run(context.Background(), env, []string{"device", "list"})
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestUnknownCommand(t *testing.T) {
	env, _, _ := testEnv(func(req *http.Request) (*http.Response, error) { return nil, nil })
	if code := Run(context.Background(), env, []string{"frobnicate"}); code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestApplyDryRunReportsPlan(t *testing.T) {
	env, out, _ := testEnv(func(req *http.Request) (*http.Response, error) {
		// Space has no existing devices, so both manifest devices are missing.
		if req.Method == http.MethodGet {
			return jsonResp(200, `{"data":{"current":1,"items":[],"last":1}}`), nil
		}
		t.Fatalf("dry-run apply must not POST: %s %s", req.Method, req.URL.Path)
		return nil, nil
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.yaml")
	manifest := "spaces:\n  - id: spc_1\n    devices:\n      - name: sensor-a\n      - name: sensor-b\n"
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	code := Run(context.Background(), env, []string{"apply", path, "--token", "qmt_x.y", "--dry-run"})
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "sensor-a") || !strings.Contains(got, "would-create") {
		t.Fatalf("stdout = %s", got)
	}
}
