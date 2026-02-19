package api

import (
	"archive/zip"
	"bytes"
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

type importResult struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Builtin     bool     `json:"builtin"`
	Keywords    []string `json:"keywords"`
}

func (s *Server) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	// SEC-11: Limit request body
	r.Body = http.MaxBytesReader(w, r.Body, maxZIPUploadSize)

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

	var imported []importResult
	for _, root := range roots {
		result, err := s.importTemplateFromZipDir(zr, root)
		if err != nil {
			log.Printf("[templates] import: failed for root %q: %v", root, err)
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		imported = append(imported, *result)
	}

	log.Printf("[templates] imported %d template(s) from ZIP", len(imported))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imported)
}

// findTemplateRoots discovers all directories containing manifest.json in the ZIP.
// Uses path (not filepath) because ZIP entries always use forward slashes.
func findTemplateRoots(zr *zip.Reader) []string {
	seen := make(map[string]bool)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Skip macOS resource forks and hidden files
		if strings.Contains(f.Name, "__MACOSX") || strings.HasPrefix(path.Base(f.Name), ".") {
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
		if f.FileInfo().IsDir() {
			continue
		}
		// Skip macOS resource forks and hidden files
		if strings.Contains(f.Name, "__MACOSX") || strings.HasPrefix(path.Base(f.Name), ".") {
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

func (s *Server) importTemplateFromZipDir(zr *zip.Reader, prefix string) (*importResult, error) {
	files := filesInPrefix(zr, prefix)

	var manifestData []byte
	var binaryFile *zip.File

	for _, f := range files {
		base := path.Base(f.Name)
		if base == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
			manifestData, err = io.ReadAll(io.LimitReader(rc, 1<<20))
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest.json")
			}
		} else if binaryFile == nil {
			binaryFile = f
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("no manifest.json found in %q", prefix)
	}
	if binaryFile == nil {
		return nil, fmt.Errorf("no binary file found alongside manifest.json in %q", prefix)
	}

	manifest, err := template.ParseManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest.json: %v", err)
	}
	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest.json must have a 'name' field")
	}

	// SEC-06: Validate template name
	name := filepath.Base(manifest.Name)
	if name != manifest.Name || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid template name in manifest")
	}

	if template.IsOfficial(name) {
		return nil, fmt.Errorf("cannot overwrite built-in template")
	}

	// SEC-06: Verify resolved path is within TemplatesDir
	tplDir := filepath.Join(s.manager.TemplatesDir, name)
	absTemplatesDir, _ := filepath.Abs(s.manager.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("template directory escapes base")
	}

	if err := os.MkdirAll(tplDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create template directory")
	}

	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), manifestData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest")
	}

	binaryName := fmt.Sprintf("presto-template-%s", name)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	rc, err := binaryFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to read binary from ZIP")
	}
	// SEC-13: Limit binary size to 100MB
	binData, err := io.ReadAll(io.LimitReader(rc, 100<<20))
	rc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read binary")
	}

	binaryPath := filepath.Join(tplDir, binaryName)
	if err := os.WriteFile(binaryPath, binData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write binary")
	}

	log.Printf("[templates] imported template %q from ZIP (prefix=%q)", name, prefix)

	return &importResult{
		Name:        manifest.Name,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Builtin:     false,
		Keywords:    manifest.Keywords,
	}, nil
}
