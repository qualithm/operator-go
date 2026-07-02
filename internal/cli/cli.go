// Package cli implements the qualithm operator CLI: verbs over the platform
// management API (authorities, enrollments, credentials, devices, api-tokens)
// plus an idempotent `apply` for device-as-code manifests.
//
// Every command authenticates with a member API token and supports two output
// modes — human tables by default, stable line-delimited JSON with --json — and
// a --dry-run that reports the planned mutation without applying it.
//
// The CLI deliberately uses the stdlib flag package; common flags (--url,
// --token, --json, --dry-run) are registered per leaf command via addCommon.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	operator "github.com/qualithm/operator-go"
)

// Exit codes. These are part of the CLI contract and must stay stable so
// scripts and agents can branch on them.
const (
	ExitOK          = 0 // success (including dry-run)
	ExitError       = 1 // transport or unexpected error
	ExitUsage       = 2 // bad flags or arguments
	ExitAuth        = 3 // 401 / 403
	ExitNotFound    = 4 // 404
	ExitConflict    = 5 // 409
	ExitRateLimited = 6 // 429
	ExitAPI         = 7 // other non-2xx response
)

// Env bundles the I/O streams and the client constructor the CLI depends on.
// main() wires os.Std*; tests inject buffers and a fake client.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// NewClient builds an API client. Defaults to operator.New; tests override
	// it to return a client backed by a fake transport.
	NewClient func(token string, opts ...operator.Option) (*operator.Client, error)
}

// DefaultEnv returns an Env wired to stdlib defaults and the real client.
func DefaultEnv() Env {
	return Env{
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		NewClient: operator.New,
	}
}

// Run dispatches args (excluding argv[0]) to the matching command and returns
// the process exit code.
func Run(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(env.Stderr, usageText)
		return ExitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		_, _ = fmt.Fprint(env.Stdout, usageText)
		return ExitOK
	case "authority", "authorities":
		return runAuthority(ctx, env, args[1:])
	case "enrollment", "enrollments":
		return runEnrollment(ctx, env, args[1:])
	case "credential", "credentials":
		return runCredential(ctx, env, args[1:])
	case "device", "devices":
		return runDevice(ctx, env, args[1:])
	case "token", "tokens", "api-token":
		return runToken(ctx, env, args[1:])
	case "apply":
		return runApply(ctx, env, args[1:])
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown command %q\n\n%s", args[0], usageText)
		return ExitUsage
	}
}

// common holds the flags shared by every leaf command.
type common struct {
	url    string
	token  string
	json   bool
	dryRun bool
}

// addCommon registers the shared flags on fs and returns a handle read after
// fs.Parse. Defaults fall back to environment variables.
func addCommon(fs *flag.FlagSet) *common {
	c := &common{}
	fs.StringVar(&c.url, "url", os.Getenv("QUALITHM_API_URL"), "management API base URL (env QUALITHM_API_URL)")
	fs.StringVar(&c.token, "token", os.Getenv("QUALITHM_API_TOKEN"), "member API token (env QUALITHM_API_TOKEN)")
	fs.BoolVar(&c.json, "json", false, "emit JSON instead of a human table")
	fs.BoolVar(&c.dryRun, "dry-run", false, "report the planned change without applying it")
	return c
}

// parseArgs parses args into fs, permuting flags and positional arguments so
// that flags may appear before or after positionals (stdlib flag otherwise
// stops at the first non-flag). It returns the collected positionals and a
// negative code on success, or an exit code to return otherwise.
func parseArgs(env Env, fs *flag.FlagSet, args []string) ([]string, int) {
	fs.SetOutput(env.Stderr)
	var positionals []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, ExitOK
			}
			return nil, ExitUsage
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		positionals = append(positionals, rest[0])
		rest = rest[1:]
	}
	return positionals, -1
}

// parseNone parses flag-only commands, discarding any positionals.
func parseNone(env Env, fs *flag.FlagSet, args []string) int {
	_, code := parseArgs(env, fs, args)
	return code
}

// parseOne parses a command that takes a single positional argument (e.g. an
// id), returning that argument. Missing positionals yield "".
func parseOne(env Env, fs *flag.FlagSet, args []string) (string, int) {
	pos, code := parseArgs(env, fs, args)
	if code >= 0 {
		return "", code
	}
	return argAt(pos, 0), -1
}

// client builds an operator client from the common flags, applying the dry-run
// setting. It returns an exit code (>=0) on failure.
func (c *common) client(env Env) (*operator.Client, int) {
	if c.token == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: an API token is required (set --token or QUALITHM_API_TOKEN)")
		return nil, ExitUsage
	}
	opts := []operator.Option{operator.WithDryRun(c.dryRun)}
	if c.url != "" {
		opts = append(opts, operator.WithBaseURL(c.url))
	}
	client, err := env.NewClient(c.token, opts...)
	if err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: %v\n", err)
		return nil, ExitUsage
	}
	return client, -1
}

// report renders the result of a command. On success it emits value as JSON
// (when --json) or invokes human. It maps dry-run and API errors to stable exit
// codes.
func (c *common) report(env Env, value any, human func(w io.Writer), err error) int {
	if err != nil {
		var dre *operator.DryRunError
		if errors.As(err, &dre) {
			return c.reportDryRun(env, dre)
		}
		return reportError(env, err)
	}
	if c.json {
		return emitJSON(env, value)
	}
	human(env.Stdout)
	return ExitOK
}

func (c *common) reportDryRun(env Env, dre *operator.DryRunError) int {
	if c.json {
		return emitJSON(env, map[string]any{
			"dryRun": true,
			"action": map[string]any{
				"method": dre.Action.Method,
				"path":   dre.Action.Path,
				"body":   dre.Action.Body,
			},
		})
	}
	_, _ = fmt.Fprintf(env.Stdout, "dry-run: would %s %s\n", dre.Action.Method, dre.Action.Path)
	if dre.Action.Body != nil {
		buf, err := json.MarshalIndent(dre.Action.Body, "  ", "  ")
		if err == nil {
			_, _ = fmt.Fprintf(env.Stdout, "  body: %s\n", buf)
		}
	}
	return ExitOK
}

func emitJSON(env Env, value any) int {
	enc := json.NewEncoder(env.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: encode output: %v\n", err)
		return ExitError
	}
	return ExitOK
}

// reportError maps an error to a message on stderr and a stable exit code.
func reportError(env Env, err error) int {
	var ce *operator.ClientError
	if errors.As(err, &ce) {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: %v\n", err)
		return exitForStatus(ce.StatusCode)
	}
	_, _ = fmt.Fprintf(env.Stderr, "qualithm: %v\n", err)
	return ExitError
}

func exitForStatus(status int) int {
	switch status {
	case 401, 403:
		return ExitAuth
	case 404:
		return ExitNotFound
	case 409:
		return ExitConflict
	case 429:
		return ExitRateLimited
	default:
		return ExitAPI
	}
}

// argAt returns args[i] or "" when out of range.
func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// okMessage is the JSON shape emitted by verbs that return no resource body
// (revoke, delete, update).
type okMessage struct {
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}
