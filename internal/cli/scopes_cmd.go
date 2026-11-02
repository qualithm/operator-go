package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	operator "github.com/qualithm/operator-go"
)

const scopeUsage = "usage: qualithm scope list|get <id>|create|update <id>|delete <id> [flags]"

func runScope(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, scopeUsage)
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return scopeList(ctx, env, args[1:])
	case "get", "show":
		return scopeGet(ctx, env, args[1:])
	case "create", "new":
		return scopeCreate(ctx, env, args[1:])
	case "update", "patch":
		return scopeUpdate(ctx, env, args[1:])
	case "delete", "rm":
		return scopeDelete(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, scopeUsage)
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown scope verb %q\n", args[0])
		return ExitUsage
	}
}

func scopeList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("scope list", flag.ContinueOnError)
	cf := addCommon(fs)
	zone := fs.String("zone", "", "restrict to a device zone")
	cursor := fs.String("cursor", "", "resume after a page, from the cursor it printed")
	limit := fs.Int("limit", 0, "items per page (max 100)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	var res operator.Page[operator.Scope]
	var err error
	if *zone != "" {
		res, err = client.ListZoneScopes(ctx, *zone, *cursor, *limit)
	} else {
		res, err = client.ListScopes(ctx, *cursor, *limit)
	}
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "NAME", "ZONE", "PARENT", "DEVICES")
		for _, s := range res.Items {
			if s == nil {
				continue
			}
			row(tw, s.ID, s.Name, s.Zone, dashPtr(s.ParentID), fmt.Sprint(s.DeviceTotal))
		}
		_ = tw.Flush()
		nextPage(w, res.NextCursor)
	}, err)
}

func scopeGet(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("scope get", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: scope id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.GetScope(ctx, id)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "id", res.ID)
		row(tw, "name", res.Name)
		row(tw, "zone", res.Zone)
		row(tw, "parent", dashPtr(res.ParentID))
		row(tw, "deviceTotal", fmt.Sprint(res.DeviceTotal))
		row(tw, "createdAt", when(res.CreatedAt))
		_ = tw.Flush()
	}, err)
}

func scopeCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("scope create", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "scope name (required)")
	zone := fs.String("zone", "", "device zone of a top-level scope")
	parent := fs.String("parent", "", "parent scope id; the new scope takes its zone")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *name == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --name is required")
		return ExitUsage
	}
	if *zone == "" && *parent == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: set --zone for a top-level scope or --parent for a child")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.CreateScope(ctx, operator.CreateScopeInput{Name: *name, Zone: *zone, ParentID: *parent})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "scope %s (%s) created in zone %s\n", res.ID, res.Name, res.Zone)
	}, err)
}

func scopeUpdate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("scope update", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "new scope name (required)")
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: scope id is required")
		return ExitUsage
	}
	if *name == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --name is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.UpdateScope(ctx, id, *name)
	return cf.report(env, struct{}{}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "scope %s renamed to %s\n", id, *name)
	}, err)
}

func scopeDelete(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("scope delete", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: scope id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.DeleteScope(ctx, id)
	return cf.report(env, struct{}{}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "scope %s deleted\n", id)
	}, err)
}
