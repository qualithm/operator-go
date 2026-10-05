// Command qualithm is the operator CLI for the Qualithm platform management
// API: fleet and provisioning management plus an idempotent `apply` for
// device-as-code manifests, authenticated with a member API token.
package main

import (
	"context"
	"os"

	"github.com/qualithm/operator-go/internal/cli"
)

func main() {
	cli.Version = resolvedVersion()
	os.Exit(cli.Run(context.Background(), cli.DefaultEnv(), os.Args[1:]))
}
