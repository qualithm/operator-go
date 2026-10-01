package main

import (
	"runtime/debug"
	"strings"
)

// version is the CLI version. Release builds stamp it via
//
//	-ldflags "-X main.version=1.2.3"
//
// (goreleaser, the Makefile, and dx's release workflow all do), and the
// release workflow fails if `--version` still reports "dev".
var version = "dev"

// readBuildInfo is swapped in tests to exercise the fallback.
var readBuildInfo = debug.ReadBuildInfo

// resolvedVersion returns the stamped version, falling back to the module
// version Go records in the binary. That covers `go install
// github.com/qualithm/operator-go/cmd/qualithm@v1.2.3`, which never sees ldflags.
func resolvedVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := readBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return version
}
