package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	operator "github.com/qualithm/operator-go"
)

func runCredential(ctx context.Context, env Env, args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: qualithm credential list|mint|cert|rotate|revoke [flags]")
		return ExitUsage
	}
	switch args[0] {
	case "list", "ls":
		return credentialList(ctx, env, args[1:])
	case "mint", "create":
		return credentialMint(ctx, env, args[1:])
	case "cert":
		return credentialCert(ctx, env, args[1:])
	case "rotate":
		return credentialRotate(ctx, env, args[1:])
	case "revoke", "delete", "rm":
		return credentialRevoke(ctx, env, args[1:])
	case "-h", "--help", "help":
		_, _ = fmt.Fprintln(env.Stdout, "usage: qualithm credential list|mint|cert|rotate|revoke [flags]")
		return ExitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: unknown credential verb %q\n", args[0])
		return ExitUsage
	}
}

func credentialList(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("credential list", flag.ContinueOnError)
	cf := addCommon(fs)
	device := fs.String("device", "", "device id (required)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *device == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --device is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.ListCredentials(ctx, *device)
	return cf.report(env, res, func(w io.Writer) {
		tw := newTable(w)
		row(tw, "ID", "KIND", "LABEL", "STATUS", "EXPIRES")
		for _, cr := range res {
			row(tw, cr.ID, cr.Kind, dash(cr.Label), cr.Status, dash(cr.ExpiresAt))
		}
		_ = tw.Flush()
	}, err)
}

func credentialMint(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("credential mint", flag.ContinueOnError)
	cf := addCommon(fs)
	device := fs.String("device", "", "device id (required)")
	label := fs.String("label", "", "credential label")
	expires := fs.String("expires-at", "", "ISO 8601 expiry timestamp")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *device == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --device is required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.MintCredential(ctx, *device, operator.MintCredentialInput{
		Label:     *label,
		ExpiresAt: *expires,
	})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "credential %s minted for device %s\n", res.Credential.ID, res.Credential.DeviceID)
		_, _ = fmt.Fprintf(w, "secret (shown once): %s\n", res.Secret)
	}, err)
}

func credentialCert(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("credential cert", flag.ContinueOnError)
	cf := addCommon(fs)
	device := fs.String("device", "", "device id (required)")
	csrFile := fs.String("csr-file", "", "path to a PEM-encoded CSR (required)")
	label := fs.String("label", "", "credential label")
	days := fs.Int("expires-days", 0, "certificate lifetime in days (default 30)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *device == "" || *csrFile == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --device and --csr-file are required")
		return ExitUsage
	}
	csr, err := os.ReadFile(*csrFile) // #nosec G304 -- csrFile is an operator-supplied path flag
	if err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "qualithm: read csr: %v\n", err)
		return ExitError
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.IssueCert(ctx, *device, operator.IssueCertInput{
		CSRPEM:        string(csr),
		Label:         *label,
		ExpiresInDays: *days,
	})
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "certificate %s issued for device %s\n", res.Credential.ID, res.Credential.DeviceID)
		_, _ = fmt.Fprintf(w, "fingerprint: %s\n", res.Credential.CertFingerprint)
		_, _ = fmt.Fprint(w, res.CertificatePEM)
		_, _ = fmt.Fprint(w, res.CACertificatePEM)
	}, err)
}

func credentialRotate(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("credential rotate", flag.ContinueOnError)
	cf := addCommon(fs)
	device := fs.String("device", "", "device id (required)")
	credential := fs.String("credential", "", "credential id to rotate (required)")
	label := fs.String("label", "", "new credential label")
	expires := fs.String("expires-at", "", "ISO 8601 expiry timestamp")
	revoke := fs.Bool("revoke", false, "revoke the old credential immediately")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *device == "" || *credential == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --device and --credential are required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	res, err := client.RotateCredential(ctx, *device, *credential, operator.MintCredentialInput{
		Label:     *label,
		ExpiresAt: *expires,
	}, *revoke)
	return cf.report(env, res, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "credential rotated: %s -> %s\n", res.Credential.RotatedFrom, res.Credential.ID)
		_, _ = fmt.Fprintf(w, "secret (shown once): %s\n", res.Secret)
	}, err)
}

func credentialRevoke(ctx context.Context, env Env, args []string) int {
	fs := flag.NewFlagSet("credential revoke", flag.ContinueOnError)
	cf := addCommon(fs)
	device := fs.String("device", "", "device id (required)")
	credential := fs.String("credential", "", "credential id (required)")
	if code := parseNone(env, fs, args); code >= 0 {
		return code
	}
	if *device == "" || *credential == "" {
		_, _ = fmt.Fprintln(env.Stderr, "qualithm: --device and --credential are required")
		return ExitUsage
	}
	client, code := cf.client(env)
	if code >= 0 {
		return code
	}
	err := client.RevokeCredential(ctx, *device, *credential)
	return cf.report(env, okMessage{Message: "credential revoked", ID: *credential}, func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "credential %s revoked\n", *credential)
	}, err)
}
