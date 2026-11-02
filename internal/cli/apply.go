package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"

	operator "github.com/qualithm/operator-go"
)

// manifest is a device-as-code description of desired fleet state. Reconcile
// covers the devices directly in each named scope, matched by device name.
type manifest struct {
	Scopes []manifestScope `yaml:"scopes" json:"scopes"`
}

type manifestScope struct {
	ID      string           `yaml:"id" json:"id"`
	Devices []manifestDevice `yaml:"devices" json:"devices"`
}

type manifestDevice struct {
	Name string `yaml:"name" json:"name"`
}

// applyResult is one reconcile decision, emitted in the JSON/human report.
type applyResult struct {
	Scope  string `json:"scope"`
	Device string `json:"device"`
	Action string `json:"action"` // "create" | "exists" | "would-create"
	ID     string `json:"id,omitempty"`
}

func runApply(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	cf := addCommon(fs)
	path, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if path == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: a manifest path is required")
		return ExitUsage
	}

	raw, err := os.ReadFile(path) // #nosec G304 -- path is an operator-supplied manifest argument
	if err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: read manifest: %v\n", err)
		return ExitError
	}
	var m manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: parse manifest: %v\n", err)
		return ExitUsage
	}
	if len(m.Scopes) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest declares no scopes")
		return ExitUsage
	}

	client, code := cf.client(env)
	if code >= 0 {
		return code
	}

	results, code := reconcile(ctx, env, client, m)
	if code >= 0 {
		return code
	}

	if cf.json {
		return emitJSON(env, results)
	}
	tw := newTable(env.Stdout)
	row(tw, "SCOPE", "DEVICE", "ACTION", "ID")
	for _, r := range results {
		row(tw, r.Scope, r.Device, r.Action, dash(r.ID))
	}
	_ = tw.Flush()
	return ExitOK
}

// reconcile compares the manifest against live state and creates missing
// devices (matched by name within a scope). It is idempotent: devices that
// already exist are reported as "exists" and left untouched. In dry-run mode
// creates are reported as "would-create" without mutating anything.
func reconcile(ctx context.Context, env Env, client *operator.Client, m manifest) ([]applyResult, int) {
	var results []applyResult
	for _, scope := range m.Scopes {
		if scope.ID == "" {
			_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest scope is missing an id")
			return nil, ExitUsage
		}
		existing, err := listScopeDeviceNames(ctx, client, scope.ID)
		if err != nil {
			return nil, reportError(env, err)
		}
		for _, dev := range scope.Devices {
			if dev.Name == "" {
				_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest device is missing a name")
				return nil, ExitUsage
			}
			if _, ok := existing[dev.Name]; ok {
				results = append(results, applyResult{Scope: scope.ID, Device: dev.Name, Action: "exists"})
				continue
			}
			created, err := client.CreateDevice(ctx, operator.CreateDeviceInput{ScopeID: scope.ID})
			var dre *operator.DryRunError
			switch {
			case errors.As(err, &dre):
				results = append(results, applyResult{Scope: scope.ID, Device: dev.Name, Action: "would-create"})
			case err != nil:
				return nil, reportError(env, err)
			default:
				// Name the freshly created device to match the manifest so the
				// next reconcile is a no-op.
				if uerr := client.UpdateDevice(ctx, created.ID, operator.UpdateDeviceInput{Name: dev.Name}); uerr != nil {
					return nil, reportError(env, uerr)
				}
				results = append(results, applyResult{Scope: scope.ID, Device: dev.Name, Action: "create", ID: created.ID})
			}
		}
	}
	return results, -1
}

// listScopeDeviceNames returns the set of device names directly in a scope,
// paging through all results.
func listScopeDeviceNames(ctx context.Context, client *operator.Client, scopeID string) (map[string]struct{}, error) {
	names := map[string]struct{}{}
	fetch := func(ctx context.Context, cursor string) (operator.Page[operator.Device], error) {
		return client.ListScopeDevices(ctx, scopeID, cursor, operator.WalkLimit)
	}
	for d, err := range operator.All(ctx, fetch) {
		if err != nil {
			return nil, err
		}
		if d.Name != "" {
			names[d.Name] = struct{}{}
		}
	}
	return names, nil
}
