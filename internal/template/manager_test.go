package template

import (
	"os"
	"path/filepath"
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
	os.WriteFile(filepath.Join(tplDir, "presto-template-mock"), data, 0755)

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
	os.WriteFile(filepath.Join(tplDir, "presto-template-mock"), data, 0755)

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
