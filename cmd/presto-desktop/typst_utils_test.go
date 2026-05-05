package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindTypstBinaryFromWindowsPrefersBundledExe(t *testing.T) {
	exeDir := t.TempDir()
	typstPath := filepath.Join(exeDir, "typst.exe")
	if err := os.WriteFile(typstPath, []byte("stub"), 0755); err != nil {
		t.Fatalf("write typst.exe: %v", err)
	}

	got := findTypstBinaryFrom(exeDir, "windows", func(name string) (string, error) {
		t.Fatalf("lookPath should not be called when bundled typst.exe exists, got %q", name)
		return "", errors.New("unreachable")
	})

	if got != typstPath {
		t.Fatalf("expected bundled typst.exe path %q, got %q", typstPath, got)
	}
}

func TestFindTypstBinaryFromWindowsUsesPathExe(t *testing.T) {
	want := filepath.Join("C:", "Tools", "typst.exe")

	got := findTypstBinaryFrom("", "windows", func(name string) (string, error) {
		if name == "typst.exe" {
			return want, nil
		}
		return "", exec.ErrNotFound
	})

	if got != want {
		t.Fatalf("expected PATH typst.exe %q, got %q", want, got)
	}
}

func TestFindTypstBinaryFromWindowsDoesNotBypassErrDot(t *testing.T) {
	got := findTypstBinaryFrom("", "windows", func(name string) (string, error) {
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
