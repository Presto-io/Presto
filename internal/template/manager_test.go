package template

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeTestTemplate(t *testing.T, root, dirName, manifestName, displayName string) string {
	t.Helper()
	tplDir := filepath.Join(root, dirName)
	if err := os.MkdirAll(tplDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"` + manifestName + `","displayName":"` + displayName + `","version":"0.1.0","author":"test"}`
	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	bin := createMockTemplate(t, t.TempDir())
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tplDir, templateBinaryName(manifestName)), data, 0755); err != nil {
		t.Fatal(err)
	}
	return tplDir
}

func TestManagerListTemplates(t *testing.T) {
	dir := t.TempDir()

	writeTestTemplate(t, dir, "mock", "mock", "Mock Template")

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

	writeTestTemplate(t, dir, "mock", "mock", "Mock")

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

	for _, dirName := range []string{"mock-a", "mock-b"} {
		writeTestTemplate(t, dir, dirName, "mock", "Mock")
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

func TestManagerListIncludesBuiltinOnlyTemplate(t *testing.T) {
	userDir := t.TempDir()
	builtinDir := t.TempDir()
	writeTestTemplate(t, builtinDir, "mock", "mock", "Builtin Mock")

	mgr := NewManagerWithBuiltin(userDir, builtinDir)
	templates, err := mgr.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("got %d templates, want 1", len(templates))
	}
	if templates[0].Manifest.DisplayName != "Builtin Mock" {
		t.Fatalf("displayName = %q", templates[0].Manifest.DisplayName)
	}
	if !templates[0].Builtin {
		t.Fatal("builtin-only template should be marked builtin")
	}
	if !mgr.Exists("mock") {
		t.Fatal("Exists should detect builtin template")
	}
}

func TestManagerUserTemplateShadowsBuiltinByManifestName(t *testing.T) {
	userDir := t.TempDir()
	builtinDir := t.TempDir()
	writeTestTemplate(t, builtinDir, "mock", "mock", "Builtin Mock")
	writeTestTemplate(t, userDir, "mock", "mock", "User Mock")

	mgr := NewManagerWithBuiltin(userDir, builtinDir)
	tpl, err := mgr.Get("mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tpl.Manifest.DisplayName != "User Mock" {
		t.Fatalf("displayName = %q, want User Mock", tpl.Manifest.DisplayName)
	}
	if tpl.Builtin {
		t.Fatal("user template should not be marked builtin")
	}
}

func TestManagerUninstallOverlayRevealsBuiltin(t *testing.T) {
	userDir := t.TempDir()
	builtinDir := t.TempDir()
	writeTestTemplate(t, builtinDir, "mock", "mock", "Builtin Mock")
	writeTestTemplate(t, userDir, "mock", "mock", "User Mock")

	mgr := NewManagerWithBuiltin(userDir, builtinDir)
	if err := mgr.Uninstall("mock"); err != nil {
		t.Fatalf("unexpected uninstall error: %v", err)
	}
	tpl, err := mgr.Get("mock")
	if err != nil {
		t.Fatalf("expected builtin after overlay deletion: %v", err)
	}
	if tpl.Manifest.DisplayName != "Builtin Mock" {
		t.Fatalf("displayName = %q, want Builtin Mock", tpl.Manifest.DisplayName)
	}
	if !tpl.Builtin {
		t.Fatal("revealed template should be builtin")
	}
}

func TestManagerUninstallBuiltinOnlyTemplateFails(t *testing.T) {
	userDir := t.TempDir()
	builtinDir := t.TempDir()
	writeTestTemplate(t, builtinDir, "mock", "mock", "Builtin Mock")

	mgr := NewManagerWithBuiltin(userDir, builtinDir)
	err := mgr.Uninstall("mock")
	if err == nil {
		t.Fatal("expected builtin uninstall to fail")
	}
	if !strings.Contains(err.Error(), "cannot remove builtin template") {
		t.Fatalf("error = %q, want cannot remove builtin template", err.Error())
	}
	tpl, getErr := mgr.Get("mock")
	if getErr != nil {
		t.Fatalf("builtin template should remain: %v", getErr)
	}
	if tpl.Manifest.DisplayName != "Builtin Mock" {
		t.Fatalf("displayName = %q", tpl.Manifest.DisplayName)
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
	if !writeManifest {
		t.Fatal("Windows install should write manifest.json")
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

func TestReplaceTemplateDirSwapsStagedDirectory(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "mock")
	stageDir := filepath.Join(dir, ".install-mock")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := replaceTemplateDir(targetDir, stageDir); err != nil {
		t.Fatalf("replaceTemplateDir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "new.txt")); err != nil {
		t.Fatalf("new staged file missing after replace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old live file should be gone after replace, stat err = %v", err)
	}
	if _, err := os.Stat(stageDir); !os.IsNotExist(err) {
		t.Fatalf("stage dir should be moved, stat err = %v", err)
	}
}

func TestReplaceTemplateDirRejectsSymlinkWithoutDeletingExisting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows setups")
	}

	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	targetDir := filepath.Join(dir, "mock")
	stageDir := filepath.Join(dir, ".install-mock")

	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, targetDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := replaceTemplateDir(targetDir, stageDir); err == nil {
		t.Fatal("replaceTemplateDir should reject symlink targets")
	}
	if _, err := os.Stat(filepath.Join(realDir, "old.txt")); err != nil {
		t.Fatalf("existing template content should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, "new.txt")); err != nil {
		t.Fatalf("stage dir should remain for caller cleanup: %v", err)
	}
}
