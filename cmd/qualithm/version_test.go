package main

import "testing"

func TestResolvedVersionPrefersStamp(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	version = "4.5.6"
	if got := resolvedVersion(); got != "4.5.6" {
		t.Errorf("resolvedVersion() = %q, want %q", got, "4.5.6")
	}
}
