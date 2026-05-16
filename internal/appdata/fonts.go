package appdata

import (
	"os"
	"path/filepath"
)

// ResolveFontPaths returns the font directories passed to Typst.
// FONT_PATHS adds extra directories using the platform path-list separator
// (semicolon on Windows, colon on Unix). The app data fonts directory remains
// included so Docker and desktop installs keep the same default behavior.
func ResolveFontPaths(defaultFontsDir string) []string {
	seen := map[string]bool{}
	var paths []string
	add := func(path string) {
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		if seen[clean] {
			return
		}
		seen[clean] = true
		paths = append(paths, clean)
	}

	add(defaultFontsDir)
	for _, path := range filepath.SplitList(os.Getenv("FONT_PATHS")) {
		add(path)
	}
	return paths
}
