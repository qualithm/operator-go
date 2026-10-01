package cli

import "fmt"

// Version is the CLI binary version. It defaults to "dev"; cmd/qualithm sets it
// at startup from main.version, which release builds stamp via the linker.
var Version = "dev"

// runVersion prints the binary version and returns [ExitOK]. It takes no flags
// and ignores any arguments so it works as a dependency-free health check.
func runVersion(env Env) int {
	_, _ = fmt.Fprintf(env.Stdout, "qualithm %s\n", Version)
	return ExitOK
}
