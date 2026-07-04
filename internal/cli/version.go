package cli

import "fmt"

// Version is the CLI binary version. It defaults to "dev" for local builds and
// is overridden at release time via the linker:
//
//	-ldflags "-X github.com/qualithm/operator-go/internal/cli.Version=1.2.3"
var Version = "dev"

// runVersion prints the binary version and returns [ExitOK]. It takes no flags
// and ignores any arguments so it works as a dependency-free health check.
func runVersion(env Env) int {
	_, _ = fmt.Fprintf(env.Stdout, "qualithm %s\n", Version)
	return ExitOK
}
