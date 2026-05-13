package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mrered/presto/internal/preview"
)

func TestPreviewRunnerWritesSessionMainTyp(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	mainTypPath, cleanup, err := runner.writeSessionFile("", "#show: doc")
	if err != nil {
		t.Fatalf("write session file: %v", err)
	}
	defer cleanup()

	if filepath.Base(mainTypPath) != "main.typ" {
		t.Fatalf("main typ path = %q, want main.typ", mainTypPath)
	}

	data, err := os.ReadFile(mainTypPath)
	if err != nil {
		t.Fatalf("read main.typ: %v", err)
	}
	if string(data) != "#show: doc" {
		t.Fatalf("main.typ = %q", string(data))
	}
	if runner.sessionWorkDir == "" {
		t.Fatal("runner should track owned session workdir")
	}
}

func TestPreviewRunnerBuildsTinymistArgs(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	got := runner.buildTinymistArgs("/tmp/main.typ", 23625, 23626)
	want := []string{
		"preview",
		"/tmp/main.typ",
		"--no-open",
		"--partial-rendering=true",
		"--data-plane-host=127.0.0.1:23625",
		"--control-plane-host=127.0.0.1:23626",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestPreviewRunnerRetryDelayCapsAtFive(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	want := []time.Duration{
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}

	for i, expected := range want {
		got, ok := runner.nextRetryDelay()
		if !ok {
			t.Fatalf("retry %d unexpectedly capped", i+1)
		}
		if got != expected {
			t.Fatalf("retry %d delay = %s, want %s", i+1, got, expected)
		}
	}

	if got, ok := runner.nextRetryDelay(); ok || got != 0 {
		t.Fatalf("retry 6 = (%s, %v), want capped", got, ok)
	}
}

func TestPreviewRunnerStopIsIdempotent(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	workDir := t.TempDir()
	ownedDir := filepath.Join(workDir, "owned")
	if err := os.Mkdir(ownedDir, 0755); err != nil {
		t.Fatalf("create owned dir: %v", err)
	}
	runner.sessionWorkDir = ownedDir

	if err := runner.stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if _, err := os.Stat(ownedDir); !os.IsNotExist(err) {
		t.Fatalf("owned dir should be removed, stat err = %v", err)
	}
	if err := runner.stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestPreviewRunnerStopsOldSessionBeforeNewDocumentIdentity(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), "tinymist")
	oldMainTypPath, oldCleanup, err := runner.writeSessionFile("", "#let doc = old")
	if err != nil {
		t.Fatalf("write old session: %v", err)
	}
	defer oldCleanup()
	oldWorkDir := runner.sessionWorkDir
	if oldWorkDir == "" {
		t.Fatal("old session workdir should be tracked")
	}

	if err := runner.stop(); err != nil {
		t.Fatalf("stop old session: %v", err)
	}
	if _, err := os.Stat(oldMainTypPath); !os.IsNotExist(err) {
		t.Fatalf("old session main.typ should be removed before new session, stat err = %v", err)
	}

	newMainTypPath, newCleanup, err := runner.writeSessionFile("", "#let doc = new")
	if err != nil {
		t.Fatalf("write new session: %v", err)
	}
	defer newCleanup()
	if runner.sessionWorkDir == "" {
		t.Fatal("new session workdir should be tracked")
	}
	if runner.sessionWorkDir == oldWorkDir {
		t.Fatalf("new session reused old workdir %q after document identity switch", oldWorkDir)
	}
	if newMainTypPath == oldMainTypPath {
		t.Fatalf("new session retained old main.typ path %q", newMainTypPath)
	}
	if _, err := os.Stat(newMainTypPath); err != nil {
		t.Fatalf("new session main.typ should exist: %v", err)
	}
}

func TestPreviewRunnerStartTinymistReturnsStartError(t *testing.T) {
	runner := newPreviewRunner(preview.NewService(), filepath.Join(t.TempDir(), "missing-tinymist"))
	if err := runner.startTinymist(context.Background(), "/tmp/main.typ", 1, 2); err == nil {
		t.Fatal("startTinymist should return start error for missing binary")
	}
}
