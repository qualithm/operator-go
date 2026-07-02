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

// manifest is a device-as-code description of desired fleet state. Reconcile is
// currently scoped to devices within named spaces, matched by device name.
type manifest struct {
	Spaces []manifestSpace `yaml:"spaces" json:"spaces"`
}

type manifestSpace struct {
	ID      string           `yaml:"id" json:"id"`
	Devices []manifestDevice `yaml:"devices" json:"devices"`
}

type manifestDevice struct {
	Name string `yaml:"name" json:"name"`
}

// applyResult is one reconcile decision, emitted in the JSON/human report.
type applyResult struct {
	Space  string `json:"space"`
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
	if len(m.Spaces) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest declares no spaces")
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
	row(tw, "SPACE", "DEVICE", "ACTION", "ID")
	for _, r := range results {
		row(tw, r.Space, r.Device, r.Action, dash(r.ID))
	}
	_ = tw.Flush()
	return ExitOK
}

// reconcile compares the manifest against live state and creates missing
// devices (matched by name within a space). It is idempotent: devices that
// already exist are reported as "exists" and left untouched. In dry-run mode
// creates are reported as "would-create" without mutating anything.
func reconcile(ctx context.Context, env Env, client *operator.Client, m manifest) ([]applyResult, int) {
	var results []applyResult
	for _, space := range m.Spaces {
		if space.ID == "" {
			_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest space is missing an id")
			return nil, ExitUsage
		}
		existing, err := listSpaceDeviceNames(ctx, client, space.ID)
		if err != nil {
			return nil, reportError(env, err)
		}
		for _, dev := range space.Devices {
			if dev.Name == "" {
				_, _ = fmt.Fprintln(env.Stderr, "qualithm: manifest device is missing a name")
				return nil, ExitUsage
			}
			if _, ok := existing[dev.Name]; ok {
				results = append(results, applyResult{Space: space.ID, Device: dev.Name, Action: "exists"})
				continue
			}
			created, err := client.CreateDevice(ctx, operator.CreateDeviceInput{SpaceID: space.ID})
			var dre *operator.DryRunError
			switch {
			case errors.As(err, &dre):
				results = append(results, applyResult{Space: space.ID, Device: dev.Name, Action: "would-create"})
			case err != nil:
				return nil, reportError(env, err)
			default:
				// Name the freshly created device to match the manifest so the
				// next reconcile is a no-op.
				if uerr := client.UpdateDevice(ctx, created.ID, operator.UpdateDeviceInput{Name: dev.Name}); uerr != nil {
					return nil, reportError(env, uerr)
				}
				results = append(results, applyResult{Space: space.ID, Device: dev.Name, Action: "create", ID: created.ID})
			}
		}
	}
	return results, -1
}

// listSpaceDeviceNames returns the set of device names in a space, paging
// through all results.
func listSpaceDeviceNames(ctx context.Context, client *operator.Client, spaceID string) (map[string]struct{}, error) {
	names := map[string]struct{}{}
	for page := 1; ; page++ {
		res, err := client.ListSpaceDevices(ctx, spaceID, page, 100)
		if err != nil {
			return nil, err
		}
		for _, d := range res.Items {
			if d != nil && d.Name != "" {
				names[d.Name] = struct{}{}
			}
		}
		if res.Last == 0 || page >= res.Last {
			break
		}
	}
	return names, nil
}
