package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	operator "github.com/qualithm/operator-go"
)

func runSpace(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm space list|get <id>|create|update <id>|delete <id> [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return spaceList(ctx, env, args[1:])
	case "get", "show":
		return spaceGet(ctx, env, args[1:])
	case "create", "new":
		return spaceCreate(ctx, env, args[1:])
	case "update", "patch":
		return spaceUpdate(ctx, env, args[1:])
	case "delete", "rm":
		return spaceDelete(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm space list|get <id>|create|update <id>|delete <id> [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown space verb %q\n", args[0])
		return ExitUsage
	}
}

func spaceList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("space list", flag.ContinueOnError)
	cf := addCommon(fs)
	page := fs.Int("page", 0, "page number")
	limit := fs.Int("limit", 0, "items per page (max 100)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.ListSpaces(ctx, *page, *limit)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "NAME", "ZONE", "DEVICES")
		for _, s := range res.Items {
			if s == nil {
				continue
			}
			row(tw, s.ID, s.Name, s.Zone, fmt.Sprint(s.DeviceTotal))
		}
		_ = tw.Flush()
	}, err)
}

func spaceGet(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("space get", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: space id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.GetSpace(ctx, id)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "id", res.ID)
		row(tw, "name", res.Name)
		row(tw, "zone", res.Zone)
		row(tw, "deviceTotal", fmt.Sprint(res.DeviceTotal))
		row(tw, "createdAt", dash(res.CreatedAt))
		_ = tw.Flush()
	}, err)
}

func spaceCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("space create", flag.ContinueOnError)
	cf := addCommon(fs)
	zone := fs.String("zone", "", "device zone (required)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *zone == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --zone is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.CreateSpace(ctx, operator.CreateSpaceInput{Zone: *zone})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "space %s (%s) created in zone %s\n", res.ID, res.Name, res.Zone)
	}, err)
}

func spaceUpdate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("space update", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "new space name (required)")
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: space id is required")
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
	err := client.UpdateSpace(ctx, id, *name)
	return cf.report(env, struct{}{}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "space %s renamed to %s\n", id, *name)
	}, err)
}

func spaceDelete(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("space delete", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: space id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.DeleteSpace(ctx, id)
	return cf.report(env, struct{}{}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "space %s deleted\n", id)
	}, err)
}
