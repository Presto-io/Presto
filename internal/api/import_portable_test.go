package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrered/presto/internal/template"
)

func makeTemplateZip(t *testing.T, root string, name string, binary []byte, extra map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeZipFile := func(name string, data []byte) {
		t.Helper()
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	prefix := root
	if prefix != "" {
		prefix += "/"
	}
	writeZipFile(prefix+"manifest.json", []byte(`{"name":"`+name+`","displayName":"`+name+`","version":"1.0.0","author":"Presto-io"}`))
	writeZipFile(prefix+"presto-template-"+name, binary)
	for name, data := range extra {
		writeZipFile(name, data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func registryWithTemplate(name string, trust string, sha string) *template.Registry {
	return &template.Registry{
		Version: 1,
		Templates: []template.RegistryEntry{
			{
				Name:  name,
				Trust: trust,
				Platforms: map[string]template.RegistryPlatformInfo{
					template.Platform(): {
						URL:    "https://example.invalid/" + name,
						SHA256: sha,
					},
				},
			},
		},
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestPortableOfficialZipWithMatchingHashInstallsToUserRoot(t *testing.T) {
	binary := []byte("official template binary")
	zipData := makeTemplateZip(t, "gongwen", "gongwen", binary, nil)
	userDir := t.TempDir()
	builtinDir := t.TempDir()
	mgr := template.NewManagerWithBuiltin(userDir, builtinDir)

	result, err := ProcessBatchZipWithOptions(zipData, mgr, nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", sha256Hex(binary)),
	})
	if err != nil {
		t.Fatalf("portable import failed: %v", err)
	}
	if len(result.Templates) != 1 {
		t.Fatalf("imported templates = %d, want 1", len(result.Templates))
	}
	if _, err := os.Stat(filepath.Join(userDir, "gongwen", "manifest.json")); err != nil {
		t.Fatalf("manifest should be installed in user root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(builtinDir, "gongwen")); !os.IsNotExist(err) {
		t.Fatalf("builtin root should not be written, stat err = %v", err)
	}
}

func TestPortableZipUnknownTemplateReturnsOfficialTemplateError(t *testing.T) {
	binary := []byte("unknown template binary")
	zipData := makeTemplateZip(t, "unknown", "unknown", binary, nil)
	mgr := template.NewManager(t.TempDir())

	_, err := ProcessBatchZipWithOptions(zipData, mgr, nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", sha256Hex(binary)),
	})
	if err == nil {
		t.Fatal("expected unknown portable template to fail")
	}
	if !strings.Contains(err.Error(), "official template") {
		t.Fatalf("error = %q, want official template", err.Error())
	}
}

func TestPortableZipHashMismatchReturnsChecksumError(t *testing.T) {
	binary := []byte("tampered template binary")
	zipData := makeTemplateZip(t, "gongwen", "gongwen", binary, nil)
	mgr := template.NewManager(t.TempDir())

	_, err := ProcessBatchZipWithOptions(zipData, mgr, nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", strings.Repeat("0", 64)),
	})
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "checksum") {
		t.Fatalf("error = %q, want checksum", err.Error())
	}
}

func TestPortableZipRejectsRuntimeUpdatePayload(t *testing.T) {
	binary := []byte("official template binary")
	zipData := makeTemplateZip(t, "gongwen", "gongwen", binary, map[string][]byte{
		"tinymist": []byte("runtime payload"),
	})
	mgr := template.NewManager(t.TempDir())

	_, err := ProcessBatchZipWithOptions(zipData, mgr, nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", sha256Hex(binary)),
	})
	if err == nil {
		t.Fatal("expected runtime payload to fail")
	}
	if !strings.Contains(err.Error(), "runtime updates are not supported") {
		t.Fatalf("error = %q, want runtime updates are not supported", err.Error())
	}
}
