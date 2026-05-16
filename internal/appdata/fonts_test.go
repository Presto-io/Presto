package appdata

import (
	"path/filepath"
	"testing"
)

func TestResolveFontPathsIncludesDefaultAndEnv(t *testing.T) {
	defaultDir := filepath.Join(t.TempDir(), "fonts")
	extraDir := filepath.Join(t.TempDir(), "extra-fonts")
	t.Setenv("FONT_PATHS", extraDir)

	got := ResolveFontPaths(defaultDir)
	if len(got) != 2 {
		t.Fatalf("ResolveFontPaths returned %d paths, want 2: %#v", len(got), got)
	}
	if got[0] != filepath.Clean(defaultDir) {
		t.Fatalf("first path = %q, want default %q", got[0], filepath.Clean(defaultDir))
	}
	if got[1] != filepath.Clean(extraDir) {
		t.Fatalf("second path = %q, want env path %q", got[1], filepath.Clean(extraDir))
	}
}

func TestResolveFontPathsDeduplicates(t *testing.T) {
	defaultDir := filepath.Join(t.TempDir(), "fonts")
	t.Setenv("FONT_PATHS", defaultDir)

	got := ResolveFontPaths(defaultDir)
	if len(got) != 1 {
		t.Fatalf("ResolveFontPaths returned %#v, want one deduplicated path", got)
	}
}
