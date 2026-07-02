package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	operator "github.com/qualithm/operator-go"
)

func runAuthority(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm authority list|create|revoke <id> [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return authorityList(ctx, env, args[1:])
	case "create", "new", "register":
		return authorityCreate(ctx, env, args[1:])
	case "revoke", "delete", "rm":
		return authorityRevoke(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm authority list|create|revoke <id> [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown authority verb %q\n", args[0])
		return ExitUsage
	}
}

func authorityList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("authority list", flag.ContinueOnError)
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
	res, err := client.ListAuthorities(ctx, *page, *limit)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "NAME", "KIND", "STATUS", "FINGERPRINT")
		for _, a := range res.Items {
			if a == nil {
				continue
			}
			row(tw, a.ID, dash(a.Name), a.Kind, a.Status, dash(a.Fingerprint))
		}
		_ = tw.Flush()
	}, err)
}

func authorityCreate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("authority create", flag.ContinueOnError)
	cf := addCommon(fs)
	name := fs.String("name", "", "authority name (required)")
	kind := fs.String("kind", "platform", "authority kind: platform|byo")
	certFile := fs.String("cert-file", "", "PEM certificate file (required for byo)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *name == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --name is required")
		return ExitUsage
	}
	in := operator.CreateAuthorityInput{Name: *name, Kind: *kind}
	if *kind == "byo" {
		if *certFile == "" {
			_, _ = fmt.Fprintln(env.Stderr, "qualithm: --cert-file is required for byo authorities")
			return ExitUsage
		}
		pem, err := os.ReadFile(*certFile) // #nosec G304 -- certFile is an operator-supplied path flag
		if err != nil {
			_, _ = fmt.Fprintf(env.Stderr, "qualithm: read cert: %v\n", err)
			return ExitError
		}
		in.CertificatePEM = string(pem)
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.CreateAuthority(ctx, in)
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "authority %s (%s) registered\n", res.ID, res.Name)
	}, err)
}

func authorityRevoke(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("authority revoke", flag.ContinueOnError)
	cf := addCommon(fs)
	id, code := parseOne(env, fs, args)
	if code >= 0 {
		return code
	}
	if id == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: authority id is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.RevokeAuthority(ctx, id)
	return cf.report(env, okMessage{Message: "authority revoked", ID: id}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "authority %s revoked\n", id)
	}, err)
}
