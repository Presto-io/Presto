package appdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDirsHonorsEnvOverrides(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PRESTO_CONFIG_DIR", filepath.Join(root, "config"))
	t.Setenv("PRESTO_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("PRESTO_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("PRESTO_LOG_DIR", filepath.Join(root, "logs"))
	t.Setenv("PRESTO_LEGACY_DIR", filepath.Join(root, "legacy"))

	dirs, err := ResolveDirs()
	if err != nil {
		t.Fatalf("ResolveDirs returned error: %v", err)
	}
	if dirs.ConfigDir != filepath.Join(root, "config") {
		t.Fatalf("ConfigDir = %q", dirs.ConfigDir)
	}
	if dirs.DataDir != filepath.Join(root, "data") {
		t.Fatalf("DataDir = %q", dirs.DataDir)
	}
	if dirs.CacheDir != filepath.Join(root, "cache") {
		t.Fatalf("CacheDir = %q", dirs.CacheDir)
	}
	if dirs.LogDir != filepath.Join(root, "logs") {
		t.Fatalf("LogDir = %q", dirs.LogDir)
	}
	if dirs.LegacyDir != filepath.Join(root, "legacy") {
		t.Fatalf("LegacyDir = %q", dirs.LegacyDir)
	}
}

func TestMigrateLegacyOnceMovesManagedDataAndRecordsMarker(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, ".presto")
	dirs := Dirs{
		ConfigDir: filepath.Join(root, "config"),
		DataDir:   filepath.Join(root, "data"),
		CacheDir:  filepath.Join(root, "cache"),
		LogDir:    filepath.Join(root, "logs"),
		LegacyDir: legacyDir,
	}

	writeTestFile(t, filepath.Join(legacyDir, "templates", "demo", "manifest.json"), []byte("{}"))
	writeTestFile(t, filepath.Join(legacyDir, "fonts", "demo.ttf"), []byte("font"))
	writeTestFile(t, filepath.Join(legacyDir, "registry-cache.json"), []byte("cache"))
	writeTestFile(t, filepath.Join(legacyDir, "notes.md"), []byte("user file"))

	result, err := MigrateLegacyOnce(dirs)
	if err != nil {
		t.Fatalf("MigrateLegacyOnce returned error: %v", err)
	}
	if !result.Attempted || result.Skipped {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dirs.ConfigDir, legacyMigrationFile)); err != nil {
		t.Fatalf("migration marker not written: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(dirs.TemplatesDir(), "demo", "manifest.json")); err != nil || string(data) != "{}" {
		t.Fatalf("template not migrated: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(dirs.FontsDir(), "demo.ttf")); err != nil || string(data) != "font" {
		t.Fatalf("font not migrated: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(dirs.CacheDir, "registry-cache.json")); err != nil || string(data) != "cache" {
		t.Fatalf("registry cache not migrated: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(legacyDir, "notes.md")); err != nil || string(data) != "user file" {
		t.Fatalf("unmanaged legacy file should remain: data=%q err=%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(legacyDir, "templates", "demo", "manifest.json")); err != nil || string(data) != "{}" {
		t.Fatalf("legacy managed files should be left in place: data=%q err=%v", data, err)
	}

	second, err := MigrateLegacyOnce(dirs)
	if err != nil {
		t.Fatalf("second MigrateLegacyOnce returned error: %v", err)
	}
	if !second.Skipped {
		t.Fatalf("second migration should be skipped: %+v", second)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
