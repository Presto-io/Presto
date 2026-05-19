package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFindTypstBinaryFromWindowsPrefersBundledExe(t *testing.T) {
	exeDir := t.TempDir()
	typstPath := filepath.Join(exeDir, "typst.exe")
	if err := os.WriteFile(typstPath, []byte("stub"), 0755); err != nil {
		t.Fatalf("write typst.exe: %v", err)
	}

	got := findTypstBinaryFrom(exeDir, "", "windows", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when bundled typst.exe exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != typstPath {
		t.Fatalf("expected bundled typst.exe path %q, got %q", typstPath, got)
	}
}

func TestFindTypstBinaryFromWindowsUsesDevDistExe(t *testing.T) {
	exeDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(exeDir, 0755); err != nil {
		t.Fatalf("create exe dir: %v", err)
	}
	typstPath := filepath.Join(exeDir, "..", "dist", "typst.exe")
	if err := os.MkdirAll(filepath.Dir(typstPath), 0755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(typstPath, []byte("stub"), 0755); err != nil {
		t.Fatalf("write dist typst.exe: %v", err)
	}

	got := findTypstBinaryFrom(exeDir, "", "windows", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when dev dist typst.exe exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != typstPath {
		t.Fatalf("expected dev dist typst.exe path %q, got %q", typstPath, got)
	}
}

func TestFindTypstBinaryFromWindowsUsesPathExe(t *testing.T) {
	want := filepath.Join("C:", "Tools", "typst.exe")

	got := findTypstBinaryFrom("", "", "windows", func(name string) (string, error) {
		if name == "typst.exe" {
			return want, nil
		}
		return "", exec.ErrNotFound
	})

	if got != want {
		t.Fatalf("expected PATH typst.exe %q, got %q", want, got)
	}
}

func TestFindTypstBinaryFromPackagedResourceBeatsUserDataRuntime(t *testing.T) {
	exeDir := t.TempDir()
	dataDir := t.TempDir()
	packaged := filepath.Join(exeDir, "..", "Resources", "typst")
	userRuntime := filepath.Join(dataDir, "runtimes", "typst", "v2.0.0", "darwin-"+runtime.GOARCH, "typst")
	for _, p := range []string{packaged, userRuntime} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("create dir: %v", err)
		}
		if err := os.WriteFile(p, []byte("stub"), 0755); err != nil {
			t.Fatalf("write runtime: %v", err)
		}
	}

	got := findTypstBinaryFrom(exeDir, dataDir, "darwin", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when packaged typst exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != packaged {
		t.Fatalf("expected packaged typst %q, got %q", packaged, got)
	}
}

func TestFindTypstBinaryFromUserDataRuntimeBeatsPath(t *testing.T) {
	dataDir := t.TempDir()
	userRuntime := filepath.Join(dataDir, "runtimes", "typst", "v2.0.0", runtime.GOOS+"-"+runtime.GOARCH, "typst")
	if runtime.GOOS == "windows" {
		userRuntime += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(userRuntime), 0755); err != nil {
		t.Fatalf("create runtime dir: %v", err)
	}
	if err := os.WriteFile(userRuntime, []byte("stub"), 0755); err != nil {
		t.Fatalf("write user runtime: %v", err)
	}

	got := findTypstBinaryFrom("", dataDir, runtime.GOOS, func(name string) (string, error) {
		return filepath.Join("PATH", name), nil
	})

	if got != userRuntime {
		t.Fatalf("expected user data runtime %q, got %q", userRuntime, got)
	}
}

func TestPortablePackagedRuntimeMissingReturnsError(t *testing.T) {
	err := validatePortablePackagedRuntimes(ReleaseCapabilities{PackagedRuntimes: true}, t.TempDir(), "darwin", "arm64")
	if err == nil {
		t.Fatal("expected missing packaged runtime error")
	}
	if !strings.Contains(err.Error(), "portable packaged runtime missing") {
		t.Fatalf("error = %q, want portable packaged runtime missing", err.Error())
	}
}

func TestFindTypstBinaryFromWindowsDoesNotBypassErrDot(t *testing.T) {
	got := findTypstBinaryFrom("", "", "windows", func(name string) (string, error) {
		if name == "typst.exe" {
			return `.\typst.exe`, exec.ErrDot
		}
		return "", exec.ErrNotFound
	})

	if got != "typst" {
		t.Fatalf("expected fallback to naked typst, got %q", got)
	}
}

func TestTypstBinaryCandidates(t *testing.T) {
	windows := typstBinaryCandidates("windows")
	if len(windows) != 2 || windows[0] != "typst.exe" || windows[1] != "typst" {
		t.Fatalf("unexpected windows candidates: %#v", windows)
	}

	other := typstBinaryCandidates("darwin")
	if len(other) != 1 || other[0] != "typst" {
		t.Fatalf("unexpected darwin candidates: %#v", other)
	}
}
