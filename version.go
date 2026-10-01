package operator

import (
	"runtime/debug"
	"strings"
)

// modulePath is this module's import path, used to find its own entry in the
// build info of whatever binary it is linked into.
const modulePath = "github.com/qualithm/operator-go"

// Version is the operator-go module version linked into the running binary,
// without the leading "v" (e.g. "0.2.0"), as recorded by the Go toolchain. It
// is "dev" when no module version is available — tests, `go run`, and builds
// from a checkout that isn't at a tag. It is reported in the User-Agent header
// (see [WithUserAgent]).
var Version = moduleVersion()

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	return versionFrom(info)
}

// versionFrom finds this module in info: as the main module when built as the
// qualithm CLI, otherwise as a dependency (honouring a replace).
func versionFrom(info *debug.BuildInfo) string {
	if info.Main.Path == modulePath {
		return cleanVersion(info.Main.Version)
	}
	for _, dep := range info.Deps {
		if dep.Path != modulePath {
			continue
		}
		if dep.Replace != nil {
			return cleanVersion(dep.Replace.Version)
		}
		return cleanVersion(dep.Version)
	}
	return "dev"
}

func cleanVersion(v string) string {
	if v == "" || v == "(devel)" {
		return "dev"
	}
	return strings.TrimPrefix(v, "v")
}
