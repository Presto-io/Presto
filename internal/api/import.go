package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/mrered/presto/internal/template"
)

// SEC-29: Maximum ZIP upload size (100MB)
const maxZIPUploadSize = 100 << 20

func readZipFileLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("ZIP entry exceeds maximum size of %d bytes", limit)
	}
	return data, nil
}

type importResult struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Builtin     bool     `json:"builtin"`
	Keywords    []string `json:"keywords"`
	Verified    string   `json:"verified"` // "verified" | "not_in_registry" | "pending" | "mismatch"
}

type TemplateImportOptions struct {
	OfficialOnly      bool
	AllowlistRegistry *template.Registry
}

func LoadBuiltinRegistrySnapshot(builtinTemplatesDir string) (*template.Registry, error) {
	if builtinTemplatesDir == "" {
		return nil, fmt.Errorf("builtin registry snapshot not configured")
	}
	data, err := os.ReadFile(filepath.Join(builtinTemplatesDir, "registry.json"))
	if err != nil {
		return nil, err
	}
	var reg template.Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

func portableTemplateImportOptions(builtinTemplatesDir string) TemplateImportOptions {
	reg, err := LoadBuiltinRegistrySnapshot(builtinTemplatesDir)
	if err != nil {
		log.Printf("[templates] portable registry snapshot unavailable: %v", err)
	}
	return TemplateImportOptions{
		OfficialOnly:      true,
		AllowlistRegistry: reg,
	}
}

func (s *Server) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	// SEC-11: Limit request body
	r.Body = http.MaxBytesReader(w, r.Body, maxZIPUploadSize)

	onConflict := r.URL.Query().Get("onConflict") // "overwrite", "skip", "rename", or "" (error)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("[templates] import: parse form failed: %v", err)
		writeJSONError(w, "invalid request or file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[templates] import: no file in request: %v", err)
		writeJSONError(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		writeJSONError(w, "only .zip files are accepted", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[templates] import: read file failed: %v", err)
		writeJSONError(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Printf("[templates] import: invalid zip: %v", err)
		writeJSONError(w, "invalid ZIP file", http.StatusBadRequest)
		return
	}

	// SEC-30: Reject path traversal in ZIP entries
	for _, f := range zr.File {
		if strings.Contains(f.Name, "..") {
			writeJSONError(w, "ZIP contains path traversal", http.StatusBadRequest)
			return
		}
	}

	// Find all directories containing manifest.json (supports nested + multi-template ZIPs)
	roots := findTemplateRoots(zr)
	if len(roots) == 0 {
		writeJSONError(w, "ZIP must contain at least one manifest.json", http.StatusBadRequest)
		return
	}
	rootSet := templateRootSet(roots)
	if err := rejectRuntimeUpdateEntries(zr, rootSet); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	importOptions := s.importOptions

	// Pre-scan: read manifests and detect conflicts
	type templateEntry struct {
		root     string
		manifest *template.Manifest
	}
	var entries []templateEntry
	var conflicts []string

	for _, root := range roots {
		manifest, err := readManifestFromZip(zr, root)
		if err != nil {
			log.Printf("[templates] import: manifest read error in root %q: %v", root, err)
			writeJSONError(w, "invalid template package", http.StatusBadRequest) // SEC-35
			return
		}
		entries = append(entries, templateEntry{root: root, manifest: manifest})
		if s.manager.Exists(manifest.Name) {
			conflicts = append(conflicts, manifest.Name)
		}
	}

	// If conflicts exist and no strategy specified, return 409
	if len(conflicts) > 0 && onConflict == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error":     "conflict",
			"conflicts": conflicts,
		})
		return
	}

	imported := make([]importResult, 0, len(entries))
	for _, entry := range entries {
		name := entry.manifest.Name
		statusName := name

		if s.manager.Exists(name) {
			switch onConflict {
			case "overwrite":
				statusName = name
			case "skip":
				log.Printf("[templates] import: skipping existing template %q", name)
				continue
			case "rename":
				name = s.manager.UniqueTemplateName(name)
				statusName = name
				log.Printf("[templates] import: auto-renaming to %q", name)
			default:
				writeJSONError(w, fmt.Sprintf("template %q already exists", name), http.StatusConflict)
				return
			}
		}

		result, err := importTemplateFromZipDir(zr, entry.root, name, s.manager, s.registry, importOptions)
		if err != nil {
			log.Printf("[templates] import: failed for root %q: %v", entry.root, err)
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		result.Name = statusName
		imported = append(imported, *result)
	}

	log.Printf("[templates] imported %d template(s) from ZIP", len(imported))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imported)
}

func uninstallUserTemplateIfPresent(mgr *template.Manager, name string) error {
	if mgr == nil {
		return fmt.Errorf("template manager not configured")
	}
	safeName := filepath.Base(name)
	if safeName != name || strings.Contains(name, "..") {
		return fmt.Errorf("invalid template name")
	}
	if _, err := os.Lstat(filepath.Join(mgr.TemplatesDir, safeName)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return mgr.Uninstall(safeName)
}

func templateRootSet(roots []string) map[string]bool {
	set := make(map[string]bool, len(roots))
	for _, root := range roots {
		set[root] = true
	}
	return set
}

func rejectRuntimeUpdateEntries(zr *zip.Reader, roots map[string]bool) error {
	for _, f := range zr.File {
		if skipZipEntry(f) {
			continue
		}
		base := strings.ToLower(path.Base(f.Name))
		switch base {
		case "typst", "typst.exe", "tinymist", "tinymist.exe":
			if !isInsideTemplateRoot(f.Name, roots) {
				return fmt.Errorf("runtime updates are not supported by template ZIP import")
			}
		}
	}
	return nil
}

// readManifestFromZip reads and parses manifest.json from a ZIP directory.
func readManifestFromZip(zr *zip.Reader, prefix string) (*template.Manifest, error) {
	files := filesInPrefix(zr, prefix)
	for _, f := range files {
		if path.Base(f.Name) == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
			data, err := readZipFileLimited(rc, 1<<20)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
			manifest, err := template.ParseManifest(data)
			if err != nil {
				return nil, fmt.Errorf("invalid manifest.json: %v", err)
			}
			if manifest.Name == "" {
				return nil, fmt.Errorf("manifest.json must have a 'name' field")
			}
			return manifest, nil
		}
	}
	return nil, fmt.Errorf("no manifest.json found in %q", prefix)
}

// skipZipEntry returns true for macOS resource forks and hidden files.
func skipZipEntry(f *zip.File) bool {
	return f.FileInfo().IsDir() ||
		strings.Contains(f.Name, "__MACOSX") ||
		strings.HasPrefix(path.Base(f.Name), ".")
}

// findTemplateRoots discovers all directories containing manifest.json in the ZIP.
// Uses path (not filepath) because ZIP entries always use forward slashes.
func findTemplateRoots(zr *zip.Reader) []string {
	seen := make(map[string]bool)
	for _, f := range zr.File {
		if skipZipEntry(f) {
			continue
		}
		if path.Base(f.Name) == "manifest.json" {
			dir := path.Dir(f.Name)
			if dir == "." {
				dir = ""
			}
			seen[dir] = true
		}
	}

	roots := make([]string, 0, len(seen))
	for d := range seen {
		roots = append(roots, d)
	}
	sort.Strings(roots)
	return roots
}

// filesInPrefix returns ZIP entries directly inside the given prefix directory.
func filesInPrefix(zr *zip.Reader, prefix string) []*zip.File {
	var result []*zip.File
	for _, f := range zr.File {
		if skipZipEntry(f) {
			continue
		}
		dir := path.Dir(f.Name)
		if dir == "." {
			dir = ""
		}
		if dir == prefix {
			result = append(result, f)
		}
	}
	return result
}

// importTemplateFromZipDir installs a single template from a ZIP directory.
// installName allows overriding the name from manifest (for rename-on-conflict).
// If registry is non-nil, the binary's SHA256 is verified against the registry.
func importTemplateFromZipDir(zr *zip.Reader, prefix string, installName string, mgr *template.Manager, registry *template.RegistryCache, opts TemplateImportOptions) (*importResult, error) {
	files := filesInPrefix(zr, prefix)

	var manifestData []byte

	for _, f := range files {
		base := path.Base(f.Name)
		if base == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
			manifestData, err = readZipFileLimited(rc, 1<<20)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("no manifest.json found in %q", prefix)
	}

	manifest, err := template.ParseManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest.json: %v", err)
	}
	binaryFile := findExpectedTemplateBinary(files, manifest.Name)
	if binaryFile == nil {
		return nil, fmt.Errorf("missing expected template binary %q", expectedTemplateBinaryName(manifest.Name))
	}

	// Use installName (may differ from manifest name for rename-on-conflict)
	name := installName

	// SEC-06: Validate template name
	safeName := filepath.Base(name)
	if safeName != name || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid template name in manifest")
	}

	tplDir := filepath.Join(mgr.TemplatesDir, name)
	absTemplatesDir, _ := filepath.Abs(mgr.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("template directory escapes base")
	}

	if err := os.MkdirAll(mgr.TemplatesDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create templates directory")
	}
	stageDir, err := os.MkdirTemp(mgr.TemplatesDir, ".import-"+name+"-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create staged template directory")
	}
	stageCommitted := false
	defer func() {
		if !stageCommitted {
			os.RemoveAll(stageDir)
		}
	}()

	// Update manifest name if it was renamed
	if name != manifest.Name {
		manifest.Name = name
		manifestData, err = json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to update manifest")
		}
	}

	if err := os.WriteFile(filepath.Join(stageDir, "manifest.json"), manifestData, 0600); err != nil { // SEC-45
		return nil, fmt.Errorf("failed to write manifest")
	}

	binaryName := expectedTemplateBinaryName(name)
	rc, err := binaryFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to read binary from ZIP")
	}
	// SEC-13: Limit binary size to 100MB
	binData, err := readZipFileLimited(rc, 100<<20)
	rc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read binary")
	}

	// SEC-01: Verify binary SHA256 against registry
	hash := sha256.Sum256(binData)
	actualSHA256 := hex.EncodeToString(hash[:])
	verified := template.VerifyNotInRegistry
	if opts.OfficialOnly {
		verified, err = verifyOfficialTemplateOverlay(opts.AllowlistRegistry, manifest.Name, actualSHA256)
		if err != nil {
			return nil, err
		}
	} else if registry != nil {
		verified = registry.VerifySHA256(manifest.Name, actualSHA256)
	}
	if verified == template.VerifyMismatch {
		return nil, fmt.Errorf("SHA256 mismatch for template %q: binary may have been tampered with", manifest.Name)
	}

	binaryPath := filepath.Join(stageDir, binaryName)
	if err := os.WriteFile(binaryPath, binData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write binary")
	}
	if err := replaceUserTemplateDir(tplDir, stageDir); err != nil {
		return nil, fmt.Errorf("failed to install template: %w", err)
	}
	stageCommitted = true

	log.Printf("[templates] imported template %q from ZIP (prefix=%q)", name, prefix)

	return &importResult{
		Name:        manifest.Name,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Keywords:    manifest.Keywords,
		Verified:    string(verified),
	}, nil
}

func expectedTemplateBinaryName(name string) string {
	binaryName := fmt.Sprintf("presto-template-%s", name)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	return binaryName
}

func findExpectedTemplateBinary(files []*zip.File, manifestName string) *zip.File {
	expected := expectedTemplateBinaryName(manifestName)
	for _, f := range files {
		if skipZipEntry(f) {
			continue
		}
		if path.Base(f.Name) == expected {
			return f
		}
	}
	return nil
}

func replaceUserTemplateDir(targetDir, stageDir string) error {
	info, err := os.Lstat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Rename(stageDir, targetDir)
		}
		return fmt.Errorf("stat existing template: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to replace symlink: %s", filepath.Base(targetDir))
	}
	if !info.IsDir() {
		return fmt.Errorf("existing template path is not a directory: %s", targetDir)
	}

	backupDir := filepath.Join(
		filepath.Dir(targetDir),
		fmt.Sprintf(".%s-backup-%d", filepath.Base(targetDir), os.Getpid()),
	)
	if err := os.Rename(targetDir, backupDir); err != nil {
		return fmt.Errorf("backup existing template: %w", err)
	}
	if err := os.Rename(stageDir, targetDir); err != nil {
		restoreErr := os.Rename(backupDir, targetDir)
		if restoreErr != nil {
			return fmt.Errorf("activate staged template: %w; restore failed: %v", err, restoreErr)
		}
		return fmt.Errorf("activate staged template: %w", err)
	}
	if err := os.RemoveAll(backupDir); err != nil {
		log.Printf("[templates] failed to remove replaced template backup %q: %v", backupDir, err)
	}
	return nil
}

func verifyOfficialTemplateOverlay(reg *template.Registry, name string, actualSHA256 string) (template.VerifyResult, error) {
	if reg == nil {
		return template.VerifyNotInRegistry, fmt.Errorf("official template registry snapshot is required")
	}
	for _, entry := range reg.Templates {
		if entry.Name != name {
			continue
		}
		if entry.Trust != "official" {
			return template.VerifyNotInRegistry, fmt.Errorf("template %q is not an official template", name)
		}
		info, ok := entry.Platforms[template.Platform()]
		if !ok || info.SHA256 == "" {
			return template.VerifyNotInRegistry, fmt.Errorf("official template %q is not available for this platform", name)
		}
		if info.SHA256 != actualSHA256 {
			return template.VerifyMismatch, fmt.Errorf("checksum mismatch for official template %q", name)
		}
		return template.VerifyMatched, nil
	}
	return template.VerifyNotInRegistry, fmt.Errorf("template %q is not an official template", name)
}
