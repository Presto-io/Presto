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

type zipEntry struct {
	name string
	data []byte
}

func makeTemplateZipOrdered(t *testing.T, root string, name string, binary []byte, orderedFiles []zipEntry) []byte {
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
	for _, file := range orderedFiles {
		writeZipFile(prefix+file.name, file.data)
	}
	writeZipFile(prefix+"presto-template-"+name, binary)
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

func TestZipImportRequiresExpectedTemplateBinaryName(t *testing.T) {
	zipData := makeTemplateZipOrdered(t, "gongwen", "gongwen", []byte("official template binary"), []zipEntry{
		{name: "README.md", data: []byte("not a template binary")},
	})
	mgr := template.NewManager(t.TempDir())

	result, err := ProcessBatchZipWithOptions(zipData, mgr, nil, TemplateImportOptions{})
	if err != nil {
		t.Fatalf("import with extra README should still find expected binary: %v", err)
	}
	if len(result.Templates) != 1 {
		t.Fatalf("imported templates = %d, want 1", len(result.Templates))
	}

	binaryPath := filepath.Join(mgr.TemplatesDir, "gongwen", "presto-template-gongwen")
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if string(binary) != "official template binary" {
		t.Fatalf("installed wrong binary contents: %q", string(binary))
	}
}

func TestZipImportMissingExpectedTemplateBinaryFails(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mustWrite := func(name string, data []byte) {
		t.Helper()
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("gongwen/manifest.json", []byte(`{"name":"gongwen","displayName":"gongwen","version":"1.0.0","author":"Presto-io"}`))
	mustWrite("gongwen/README.md", []byte("not a template binary"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := ProcessBatchZipWithOptions(buf.Bytes(), template.NewManager(t.TempDir()), nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", strings.Repeat("0", 64)),
	})
	if err == nil {
		t.Fatal("expected import without expected binary to fail")
	}
	if !strings.Contains(err.Error(), "missing expected template binary") {
		t.Fatalf("error = %q, want missing expected template binary", err.Error())
	}
}

func TestPortableOverwriteChecksumMismatchKeepsExistingUserTemplate(t *testing.T) {
	userDir := t.TempDir()
	existingDir := filepath.Join(userDir, "gongwen")
	if err := os.MkdirAll(existingDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "manifest.json"), []byte(`{"name":"gongwen","displayName":"existing","version":"1.0.0","author":"Presto-io"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "presto-template-gongwen"), []byte("existing binary"), 0600); err != nil {
		t.Fatal(err)
	}

	tampered := []byte("tampered template binary")
	zipData := makeTemplateZip(t, "gongwen", "gongwen", tampered, nil)
	_, err := ProcessBatchZipWithOptions(zipData, template.NewManager(userDir), nil, TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: registryWithTemplate("gongwen", "official", strings.Repeat("0", 64)),
	})
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}

	binary, readErr := os.ReadFile(filepath.Join(existingDir, "presto-template-gongwen"))
	if readErr != nil {
		t.Fatalf("existing binary should remain after failed overwrite: %v", readErr)
	}
	if string(binary) != "existing binary" {
		t.Fatalf("existing binary was replaced after failed overwrite: %q", string(binary))
	}
}
