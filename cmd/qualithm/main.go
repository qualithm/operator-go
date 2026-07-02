// Command qualithm is the operator CLI for the qualithm platform management
// API: fleet and provisioning management (authorities, enrollments,
// credentials, devices, api-tokens) plus an idempotent `apply` for
// device-as-code manifests, authenticated with a member API token.
package main

import (
	"context"
	"os"

	"github.com/qualithm/operator-go/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), cli.DefaultEnv(), os.Args[1:]))
}
