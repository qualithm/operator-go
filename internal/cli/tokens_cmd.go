package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func runToken(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm token list|revoke <id> [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return tokenList(ctx, env, args[1:])
	case "create", "new":
		// Minting is browser-only, so a leaked token can't renew itself.
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: create API tokens in the app, on your team page")
		return ExitUsage
	case "revoke", "delete", "rm":
		return tokenRevoke(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm token list|revoke <id> [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown token verb %q\n", args[0])
		return ExitUsage
	}
}

func tokenList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("token list", flag.ContinueOnError)
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
	res, err := client.ListAPITokens(ctx, *page, *limit)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "NAME", "ROLE", "STATUS", "EXPIRES", "LAST-USED")
		for _, t := range res.Items {
			if t == nil {
				continue
			}
			role := t.Role
			if role == "" {
				role = "your role"
			}
			row(tw, t.ID, dash(t.Name), role, t.Status, when(t.ExpiresAt), when(t.LastUsedAt))
		}
		_ = tw.Flush()
	}, err)
}

func tokenRevoke(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("token revoke", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: token id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.RevokeAPIToken(ctx, id)
	return cf.report(env, okMessage{Message: "api token revoked", ID: id}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "api token %s revoked\n", id)
	}, err)
}
