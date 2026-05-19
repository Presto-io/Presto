package template

import (
	"path/filepath"
	"testing"
)

func TestResolveBuiltinTemplatesDirDarwin(t *testing.T) {
	got := ResolveBuiltinTemplatesDir(filepath.Join("Presto.app", "Contents", "MacOS"), "darwin")
	want := filepath.Join("Presto.app", "Contents", "MacOS", "..", "Resources", "templates")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveBuiltinTemplatesDirWindowsLinux(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		got := ResolveBuiltinTemplatesDir(filepath.Join("dist", "presto"), goos)
		want := filepath.Join("dist", "presto", "templates")
		if got != want {
			t.Fatalf("%s got %q, want %q", goos, got, want)
		}
	}
}

func TestResolveDevBuiltinTemplatesDir(t *testing.T) {
	got := ResolveDevBuiltinTemplatesDir(filepath.Join("Presto", "build", "bin"))
	want := filepath.Join("Presto", "build", "bin", "..", "..", "template-registry", "templates")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
