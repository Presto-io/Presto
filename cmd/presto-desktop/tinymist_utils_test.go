package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindTinymistBinaryFromPrefersBundledSidecar(t *testing.T) {
	exeDir := t.TempDir()
	want := filepath.Join(exeDir, "..", "Resources", "sidecars", "tinymist", "darwin-arm64", "tinymist")
	if err := os.MkdirAll(filepath.Dir(want), 0755); err != nil {
		t.Fatalf("create sidecar dir: %v", err)
	}
	if err := os.WriteFile(want, []byte("stub"), 0755); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	got := findTinymistBinaryFrom(exeDir, "darwin", "arm64", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when bundled sidecar exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != want {
		t.Fatalf("expected bundled sidecar %q, got %q", want, got)
	}
}

func TestFindTinymistBinaryFromPrefersResourcesBeforeSidecar(t *testing.T) {
	exeDir := t.TempDir()
	resources := filepath.Join(exeDir, "..", "Resources", "tinymist")
	sidecar := filepath.Join(exeDir, "..", "Resources", "sidecars", "tinymist", "darwin-arm64", "tinymist")
	if err := os.MkdirAll(filepath.Dir(resources), 0755); err != nil {
		t.Fatalf("create resources dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(sidecar), 0755); err != nil {
		t.Fatalf("create sidecar dir: %v", err)
	}
	if err := os.WriteFile(resources, []byte("resources"), 0755); err != nil {
		t.Fatalf("write resources tinymist: %v", err)
	}
	if err := os.WriteFile(sidecar, []byte("sidecar"), 0755); err != nil {
		t.Fatalf("write sidecar tinymist: %v", err)
	}

	got := findTinymistBinaryFrom(exeDir, "darwin", "arm64", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when resources tinymist exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != resources {
		t.Fatalf("expected Resources tinymist %q, got %q", resources, got)
	}
}

func TestFindTinymistBinaryFromPrefersResourcesBeforeBesideExecutable(t *testing.T) {
	exeDir := t.TempDir()
	resources := filepath.Join(exeDir, "..", "Resources", "tinymist")
	beside := filepath.Join(exeDir, "tinymist")
	if err := os.MkdirAll(filepath.Dir(resources), 0755); err != nil {
		t.Fatalf("create resources dir: %v", err)
	}
	if err := os.WriteFile(resources, []byte("resources"), 0755); err != nil {
		t.Fatalf("write resources tinymist: %v", err)
	}
	if err := os.WriteFile(beside, []byte("beside"), 0755); err != nil {
		t.Fatalf("write beside tinymist: %v", err)
	}

	got := findTinymistBinaryFrom(exeDir, "darwin", "arm64", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when resources tinymist exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != resources {
		t.Fatalf("expected Resources tinymist %q, got %q", resources, got)
	}
}

func TestFindTinymistBinaryFromFindsDevDistBinary(t *testing.T) {
	exeDir := filepath.Join(t.TempDir(), "bin")
	tinymistPath := filepath.Join(exeDir, "..", "dist", "tinymist.exe")
	if err := os.MkdirAll(filepath.Dir(tinymistPath), 0755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(tinymistPath, []byte("stub"), 0755); err != nil {
		t.Fatalf("write dist tinymist.exe: %v", err)
	}

	got := findTinymistBinaryFrom(exeDir, "windows", "amd64", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when dev dist tinymist.exe exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != tinymistPath {
		t.Fatalf("expected dev dist tinymist.exe path %q, got %q", tinymistPath, got)
	}
}

func TestFindTinymistBinaryFromWindowsUsesPathExe(t *testing.T) {
	want := filepath.Join("C:", "Tools", "tinymist.exe")

	got := findTinymistBinaryFrom("", "windows", "amd64", func(name string) (string, error) {
		if name == "tinymist.exe" {
			return want, nil
		}
		return "", exec.ErrNotFound
	})

	if got != want {
		t.Fatalf("expected PATH tinymist.exe %q, got %q", want, got)
	}
}

func TestFindTinymistBinaryFromWindowsDoesNotBypassErrDot(t *testing.T) {
	got := findTinymistBinaryFrom("", "windows", "amd64", func(name string) (string, error) {
		if name == "tinymist.exe" {
			return `.\tinymist.exe`, exec.ErrDot
		}
		return "", exec.ErrNotFound
	})

	if got != "tinymist" {
		t.Fatalf("expected fallback to naked tinymist, got %q", got)
	}
}

func TestTinymistBinaryCandidates(t *testing.T) {
	windows := tinymistBinaryCandidates("windows")
	if len(windows) != 2 || windows[0] != "tinymist.exe" || windows[1] != "tinymist" {
		t.Fatalf("unexpected windows candidates: %#v", windows)
	}

	other := tinymistBinaryCandidates("darwin")
	if len(other) != 1 || other[0] != "tinymist" {
		t.Fatalf("unexpected darwin candidates: %#v", other)
	}
}
