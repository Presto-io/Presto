package template

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestPathJoin(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"base/dir/file", []string{"base", "dir", "file.txt"}, filepath.Join("base", "dir", "file.txt")},
		{".presto/templates/official", []string{".presto", "templates", "official"}, filepath.Join(".presto", "templates", "official")},
	}

	for _, tt := range tests {
		result := filepath.Join(tt.args...)
		if result != tt.expected {
			t.Errorf("filepath.Join(%v) = %s, want %s", tt.args, result, tt.expected)
		}
	}
}

func TestPathBase(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{filepath.Join("dir", "file.txt"), "file.txt"},
		{filepath.Join("templates", "official", "template.tar.gz"), "template.tar.gz"},
		{"file.txt", "file.txt"},
	}

	for _, tt := range tests {
		result := filepath.Base(tt.path)
		if result != tt.expected {
			t.Errorf("filepath.Base(%s) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func TestPathDir(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{filepath.Join("dir", "file.txt"), "dir"},
		{filepath.Join("templates", "official", "template.tar.gz"), filepath.Join("templates", "official")},
		{"file.txt", "."},
	}

	for _, tt := range tests {
		result := filepath.Dir(tt.path)
		if result != tt.expected {
			t.Errorf("filepath.Dir(%s) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func TestTemplatePath(t *testing.T) {
	// Test template path construction
	templatesDir := filepath.Join(".presto", "templates")
	templateName := "official-template"
	tplPath := filepath.Join(templatesDir, templateName)

	// Verify path is valid on current platform
	if !filepath.IsAbs(tplPath) {
		// Relative path is OK for tests
		t.Logf("Template path: %s", tplPath)
	}

	// Verify path components
	base := filepath.Base(tplPath)
	if base != templateName {
		t.Errorf("Base name = %s, want %s", base, templateName)
	}

	dir := filepath.Dir(tplPath)
	if dir != templatesDir {
		t.Errorf("Dir = %s, want %s", dir, templatesDir)
	}
}

func TestWindowsPathOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix only")
	}

	// Test that Windows-style paths are handled correctly on Unix
	winPath := "C:\\Users\\Alice\\.presto\\templates"

	// filepath.Join should normalize separators
	joined := filepath.Join(winPath, "template")
	t.Logf("Windows path joined on Unix: %s", joined)

	// Note: This test is informational, not asserting correctness
	// Real Windows paths should not appear on Unix systems
}

func TestUnixPathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}

	// Test that Unix-style paths are handled correctly on Windows
	unixPath := "/home/alice/.presto/templates"

	// filepath.Join should normalize separators
	joined := filepath.Join(unixPath, "template")
	t.Logf("Unix path joined on Windows: %s", joined)

	// Note: This test is informational, not asserting correctness
	// Real Unix paths should not appear on Windows systems
}
