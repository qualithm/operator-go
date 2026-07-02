package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	operator "github.com/qualithm/operator-go"
)

func runDevice(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm device list|get <id>|create|update <id>|delete <id> [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return deviceList(ctx, env, args[1:])
	case "get", "show":
		return deviceGet(ctx, env, args[1:])
	case "create", "new":
		return deviceCreate(ctx, env, args[1:])
	case "update", "patch":
		return deviceUpdate(ctx, env, args[1:])
	case "delete", "rm":
		return deviceDelete(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm device list|get <id>|create|update <id>|delete <id> [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown device verb %q\n", args[0])
		return ExitUsage
	}
}

func deviceList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("device list", flag.ContinueOnError)
	cf := addCommon(fs)
	space := fs.String("space", "", "restrict to a space id")
	page := fs.Int("page", 0, "page number")
	limit := fs.Int("limit", 0, "items per page (max 100)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	var res operator.Page[operator.Device]
	var err error
	if *space != "" {
		res, err = client.ListSpaceDevices(ctx, *space, *page, *limit)
	} else {
		res, err = client.ListDevices(ctx, *page, *limit)
	}
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "NAME", "SPACE", "SPACE-NAME")
		for _, d := range res.Items {
			if d == nil {
				continue
			}
			row(tw, d.ID, dash(d.Name), d.SpaceID, dash(d.SpaceName))
		}
		_ = tw.Flush()
	}, err)
}

func deviceGet(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("device get", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: device id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.GetDevice(ctx, id)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "id", res.ID)
		row(tw, "name", dash(res.Name))
		row(tw, "space", res.SpaceID)
		row(tw, "spaceName", dash(res.SpaceName))
		row(tw, "createdAt", dash(res.CreatedAt))
		_ = tw.Flush()
	}, err)
}

func deviceCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("device create", flag.ContinueOnError)
	cf := addCommon(fs)
	space := fs.String("space", "", "space id (required)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *space == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --space is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.CreateDevice(ctx, operator.CreateDeviceInput{SpaceID: *space})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "device %s (%s) created in space %s\n", res.ID, res.Name, res.SpaceID)
	}, err)
}

func deviceUpdate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("device update", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "new device name")
	space := fs.String("space", "", "move to this space id (same zone)")
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: device id is required")
		return ExitUsage
	}
	if *name == "" && *space == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: set --name and/or --space")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.UpdateDevice(ctx, id, operator.UpdateDeviceInput{Name: *name, SpaceID: *space})
	return cf.report(env, okMessage{Message: "device updated", ID: id}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "device %s updated\n", id)
	}, err)
}

func deviceDelete(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("device delete", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: device id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.DeleteDevice(ctx, id)
	return cf.report(env, okMessage{Message: "device deleted", ID: id}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "device %s deleted\n", id)
	}, err)
}
