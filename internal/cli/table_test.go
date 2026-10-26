package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWhen(t *testing.T) {
	t.Parallel()
	if got := when(time.Time{}); got != "-" {
		t.Fatalf("zero time = %q, want -", got)
	}
	at := time.Date(2026, 10, 1, 21, 30, 0, 0, time.FixedZone("NZDT", 13*3600))
	if got := when(at); got != "2026-10-01T08:30:00Z" {
		t.Fatalf("when = %q", got)
	}
}

func TestParseExpiresAt(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	env := Env{Stdin: strings.NewReader(""), Stdout: &bytes.Buffer{}, Stderr: &errBuf}

	if got, ok := parseExpiresAt(env, ""); !ok || got != nil {
		t.Fatalf("empty = %v %v, want nil true", got, ok)
	}
	got, ok := parseExpiresAt(env, "2026-12-31T00:00:00Z")
	if !ok || got == nil || !got.Equal(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("valid = %v %v", got, ok)
	}
	if _, ok := parseExpiresAt(env, "next tuesday"); ok {
		t.Fatal("invalid time accepted")
	}
	if !strings.Contains(errBuf.String(), "--expires-at must be an RFC 3339 time") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}
