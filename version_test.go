package operator

import (
	"runtime/debug"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "main module",
			info: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.2.0"}},
			want: "0.2.0",
		},
		{
			name: "main module without version",
			info: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}},
			want: "dev",
		},
		{
			name: "dependency",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/qualithm/operator-mcp"},
				Deps: []*debug.Module{
					{Path: "golang.org/x/net", Version: "v0.44.0"},
					{Path: modulePath, Version: "v0.2.1"},
				},
			},
			want: "0.2.1",
		},
		{
			name: "replaced dependency",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/qualithm/operator-mcp"},
				Deps: []*debug.Module{{
					Path:    modulePath,
					Version: "v0.2.1",
					Replace: &debug.Module{Path: "../operator-go"},
				}},
			},
			want: "dev",
		},
		{
			name: "absent",
			info: &debug.BuildInfo{Main: debug.Module{Path: "example.com/other"}},
			want: "dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFrom(tt.info); got != tt.want {
				t.Errorf("versionFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModuleVersionWithoutBuildInfo(t *testing.T) {
	if got := moduleVersion(func() (*debug.BuildInfo, bool) { return nil, false }); got != "dev" {
		t.Errorf("moduleVersion() = %q, want %q", got, "dev")
	}
}
