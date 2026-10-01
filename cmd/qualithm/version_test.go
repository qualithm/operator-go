package main

import (
	"runtime/debug"
	"testing"
)

func TestResolvedVersionPrefersStamp(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	version = "4.5.6"
	if got := resolvedVersion(); got != "4.5.6" {
		t.Errorf("resolvedVersion() = %q, want %q", got, "4.5.6")
	}
}

func TestResolvedVersionFallsBackToBuildInfo(t *testing.T) {
	origVersion, origRead := version, readBuildInfo
	t.Cleanup(func() { version, readBuildInfo = origVersion, origRead })
	version = "dev"

	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"tagged module", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true, "1.2.3"},
		{"devel build", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev"},
		{"no build info", nil, false, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readBuildInfo = func() (*debug.BuildInfo, bool) { return tt.info, tt.ok }
			if got := resolvedVersion(); got != tt.want {
				t.Errorf("resolvedVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
