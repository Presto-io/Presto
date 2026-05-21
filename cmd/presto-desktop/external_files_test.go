package main

import (
	"path/filepath"
	"testing"
)

func TestFilterExternalPathsSkipsFlagsAndLogFile(t *testing.T) {
	originalLogFilePath := logFilePath
	defer func() {
		logFilePath = originalLogFilePath
	}()

	dir := t.TempDir()
	logFilePath = filepath.Join(dir, "presto-log.txt")
	documentPath := filepath.Join(dir, "lesson.md")

	filtered := filterExternalPaths([]string{
		"--verbose",
		"--log-file",
		logFilePath,
		"presto://install/template",
		documentPath,
		documentPath,
	})

	if len(filtered) != 1 {
		t.Fatalf("expected only document path, got %v", filtered)
	}
	if filtered[0] != documentPath {
		t.Fatalf("expected %q, got %q", documentPath, filtered[0])
	}
}
