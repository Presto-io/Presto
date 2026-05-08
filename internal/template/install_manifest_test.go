package template

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRegistryInstallUsesRegistryManifestWithoutExecutingBinary(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	data := []byte("this is intentionally not an executable")
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	manifest := []byte(`{"name":"mock","displayName":"Mock","version":"1.0.0","author":"test"}`)

	filename := "presto-template-mock-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	err := mgr.completeInstall("owner", "repo", data, hash, &InstallOpts{
		Trust:            "official",
		TemplateName:     "mock",
		TemplateVersion:  "1.0.0",
		RegistryManifest: manifest,
	}, time.Now(), downloadCandidate{
		source:   "test",
		url:      "https://presto.c-1o.top/templates/mock/binaries/" + filename,
		filename: filename,
	})
	if err != nil {
		t.Fatalf("completeInstall() failed: %v", err)
	}

	tplDir := filepath.Join(dir, "mock")
	if got, err := os.ReadFile(filepath.Join(tplDir, "manifest.json")); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	} else if string(got) != string(manifest) {
		t.Fatalf("manifest.json = %s, want %s", got, manifest)
	}
	if _, err := os.Stat(filepath.Join(tplDir, "install-lock.json")); err != nil {
		t.Fatalf("install-lock.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tplDir, filename)); runtime.GOOS == "windows" && err != nil {
		t.Fatalf("Windows should preserve downloaded filename: %v", err)
	}
}

func TestRegistryManifestIdentityMustMatchRegistry(t *testing.T) {
	_, err := NewManager(t.TempDir()).resolveManifestForInstall(nil, &InstallOpts{
		TemplateName:     "expected",
		TemplateVersion:  "1.0.0",
		RegistryManifest: []byte(`{"name":"other","version":"1.0.0"}`),
	})
	if err == nil {
		t.Fatal("expected mismatched manifest name to fail")
	}
}
