package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	operator "github.com/qualithm/operator-go"
)

func runEnrollment(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm enrollment list|create|revoke [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return enrollmentList(ctx, env, args[1:])
	case "create", "new":
		return enrollmentCreate(ctx, env, args[1:])
	case "revoke", "delete", "rm":
		return enrollmentRevoke(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm enrollment list|create|revoke [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown enrollment verb %q\n", args[0])
		return ExitUsage
	}
}

func enrollmentList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("enrollment list", flag.ContinueOnError)
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
	res, err := client.ListEnrollments(ctx, *page, *limit)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "SPACE", "LABEL", "STATUS", "EXPIRES")
		for _, e := range res.Items {
			if e == nil {
				continue
			}
			row(tw, e.ID, e.SpaceID, dash(e.Label), e.Status, dash(e.ExpiresAt))
		}
		_ = tw.Flush()
	}, err)
}

func enrollmentCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("enrollment create", flag.ContinueOnError)
	cf := addCommon(fs)
	space := fs.String("space", "", "space id (required)")
	label := fs.String("label", "", "enrollment label")
	expires := fs.Int("expires-minutes", 0, "minutes until the code expires (default 1440)")
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
	res, err := client.CreateEnrollment(ctx, operator.CreateEnrollmentInput{
		SpaceID:          *space,
		Label:            *label,
		ExpiresInMinutes: *expires,
	})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "enrollment %s created for space %s\n", res.Enrollment.ID, res.Enrollment.SpaceID)
		_, _ = fmt.Fprintf(w, "claim code (shown once): %s\n", res.Code)
	}, err)
}

func enrollmentRevoke(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("enrollment revoke", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: enrollment id is required")
		return ExitUsage
	}
	c, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := c.RevokeEnrollment(ctx, id)
	return cf.report(env, okMessage{Message: "enrollment revoked", ID: id}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "enrollment %s revoked\n", id)
	}, err)
}
