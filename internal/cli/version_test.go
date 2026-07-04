package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			env := Env{Stdin: strings.NewReader(""), Stdout: &out, Stderr: &errBuf}
			code := Run(context.Background(), env, []string{arg})
			if code != ExitOK {
				t.Fatalf("exit = %d, stderr = %s", code, errBuf.String())
			}
			if !strings.Contains(out.String(), Version) {
				t.Fatalf("stdout %q does not contain version %q", out.String(), Version)
			}
		})
	}
}

func TestVersionNeedsNoToken(t *testing.T) {
	var out, errBuf bytes.Buffer
	env := Env{Stdin: strings.NewReader(""), Stdout: &out, Stderr: &errBuf}
	if code := Run(context.Background(), env, []string{"version"}); code != ExitOK {
		t.Fatalf("version required setup it should not: exit = %d, stderr = %s", code, errBuf.String())
	}
}
