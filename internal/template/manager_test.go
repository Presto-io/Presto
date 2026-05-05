package template

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManagerListTemplates(t *testing.T) {
	dir := t.TempDir()

	tplDir := filepath.Join(dir, "mock")
	os.MkdirAll(tplDir, 0755)

	manifest := `{"name":"mock","displayName":"Mock Template","version":"0.1.0","author":"test"}`
	os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644)

	bin := createMockTemplate(t, t.TempDir())
	data, _ := os.ReadFile(bin)
	os.WriteFile(filepath.Join(tplDir, templateBinaryName("mock")), data, 0755)

	mgr := NewManager(dir)
	templates, err := mgr.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("got %d templates, want 1", len(templates))
	}
	if templates[0].Manifest.Name != "mock" {
		t.Errorf("got name %q, want %q", templates[0].Manifest.Name, "mock")
	}
}

func TestManagerGet(t *testing.T) {
	dir := t.TempDir()

	tplDir := filepath.Join(dir, "mock")
	os.MkdirAll(tplDir, 0755)

	manifest := `{"name":"mock","displayName":"Mock","version":"0.1.0","author":"test"}`
	os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644)

	bin := createMockTemplate(t, t.TempDir())
	data, _ := os.ReadFile(bin)
	os.WriteFile(filepath.Join(tplDir, templateBinaryName("mock")), data, 0755)

	mgr := NewManager(dir)
	tpl, err := mgr.Get("mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tpl.Manifest.DisplayName != "Mock" {
		t.Errorf("got %q, want %q", tpl.Manifest.DisplayName, "Mock")
	}

	_, err = mgr.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestManagerListTemplateWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	tplDir := filepath.Join(dir, "mock")
	os.MkdirAll(tplDir, 0755)

	bin := createMockTemplate(t, t.TempDir())
	data, _ := os.ReadFile(bin)
	rawName := "presto-template-mock-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		rawName += ".exe"
	}
	rawPath := filepath.Join(tplDir, rawName)
	os.WriteFile(rawPath, data, 0755)

	mgr := NewManager(dir)
	templates, err := mgr.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("got %d templates, want 1", len(templates))
	}
	if templates[0].Manifest.Name != "mock" {
		t.Errorf("got name %q, want %q", templates[0].Manifest.Name, "mock")
	}
	if filepath.Base(templates[0].BinaryPath) != rawName {
		t.Errorf("binary path = %q, want raw filename %q", templates[0].BinaryPath, rawName)
	}
	if _, err := os.Stat(filepath.Join(tplDir, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("manifest.json should not be generated, stat err = %v", err)
	}
	if !mgr.Exists("mock") {
		t.Fatal("Exists should detect template without manifest.json")
	}
}

func TestManagerListSkipsDuplicateWithoutRenaming(t *testing.T) {
	dir := t.TempDir()
	bin := createMockTemplate(t, t.TempDir())
	data, _ := os.ReadFile(bin)

	for _, dirName := range []string{"mock-a", "mock-b"} {
		tplDir := filepath.Join(dir, dirName)
		os.MkdirAll(tplDir, 0755)
		manifest := `{"name":"mock","displayName":"Mock","version":"0.1.0","author":"test"}`
		os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644)
		os.WriteFile(filepath.Join(tplDir, templateBinaryName("mock")), data, 0755)
	}

	mgr := NewManager(dir)
	templates, err := mgr.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("got %d templates, want duplicate to be skipped without rename", len(templates))
	}
	if _, err := os.Stat(filepath.Join(dir, "mock-2")); !os.IsNotExist(err) {
		t.Fatalf("duplicate template should not be renamed to mock-2, stat err = %v", err)
	}
}

func TestInstallArtifactLayoutWindowsPreservesDownloadedFile(t *testing.T) {
	binaryName, writeManifest, err := installArtifactLayout("windows", "gongwen", "presto-template-gongwen-windows-amd64.exe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if binaryName != "presto-template-gongwen-windows-amd64.exe" {
		t.Fatalf("binaryName = %q", binaryName)
	}
	if writeManifest {
		t.Fatal("Windows install should not write manifest.json")
	}
}

func TestInstallArtifactLayoutNonWindowsKeepsExistingLayout(t *testing.T) {
	binaryName, writeManifest, err := installArtifactLayout("darwin", "gongwen", "presto-template-gongwen-darwin-arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if binaryName != "presto-template-gongwen" {
		t.Fatalf("binaryName = %q", binaryName)
	}
	if !writeManifest {
		t.Fatal("non-Windows install should keep writing manifest.json")
	}
}
