package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	operator "github.com/qualithm/operator-go"
)

func runToken(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm token list|create|revoke <id> [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return tokenList(ctx, env, args[1:])
	case "create", "new":
		return tokenCreate(ctx, env, args[1:])
	case "revoke", "delete", "rm":
		return tokenRevoke(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm token list|create|revoke <id> [flags]")
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
		row(tw, "ID", "NAME", "STATUS", "EXPIRES", "LAST-USED")
		for _, t := range res.Items {
			if t == nil {
				continue
			}
			row(tw, t.ID, dash(t.Name), t.Status, dash(t.ExpiresAt), dash(t.LastUsedAt))
		}
		_ = tw.Flush()
	}, err)
}

func tokenCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("token create", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "token name")
	days := fs.Int("expires-days", 0, "days until expiry (default 90, max 365)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.CreateAPIToken(ctx, operator.CreateAPITokenInput{
		Name:          *name,
		ExpiresInDays: *days,
	})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "api token %s created\n", res.Token.ID)
		_, _ = fmt.Fprintf(w, "secret (shown once): %s\n", res.Secret)
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
