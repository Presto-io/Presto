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
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mrered/presto/internal/template"
)

// SEC-29: Maximum ZIP upload size (100MB)
const maxZIPUploadSize = 100 << 20

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

	// Find manifest.json and binary at root level
	var manifestData []byte
	var binaryFile *zip.File
	for _, f := range zr.File {
		// SEC-30: Reject path traversal in ZIP entries
		if strings.Contains(f.Name, "..") {
			writeJSONError(w, "ZIP contains path traversal", http.StatusBadRequest)
			return
		}
		// Only look at root-level files
		if strings.Contains(f.Name, "/") {
			continue
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				writeJSONError(w, "failed to read manifest.json", http.StatusBadRequest)
				return
			}
			manifestData, err = io.ReadAll(io.LimitReader(rc, 1<<20))
			rc.Close()
			if err != nil {
				writeJSONError(w, "failed to read manifest.json", http.StatusBadRequest)
				return
			}
		} else if binaryFile == nil {
			binaryFile = f
		}
	}

	if manifestData == nil {
		writeJSONError(w, "ZIP must contain manifest.json at root", http.StatusBadRequest)
		return
	}
	if binaryFile == nil {
		writeJSONError(w, "ZIP must contain a binary file at root", http.StatusBadRequest)
		return
	}

	manifest, err := template.ParseManifest(manifestData)
	if err != nil {
		log.Printf("[templates] import: invalid manifest: %v", err)
		writeJSONError(w, "invalid manifest.json", http.StatusBadRequest)
		return
	}
	if manifest.Name == "" {
		writeJSONError(w, "manifest.json must have a 'name' field", http.StatusBadRequest)
		return
	}

	// SEC-06: Validate template name
	name := filepath.Base(manifest.Name)
	if name != manifest.Name || strings.Contains(name, "..") {
		writeJSONError(w, "invalid template name in manifest", http.StatusBadRequest)
		return
	}

	if template.IsOfficial(name) {
		writeJSONError(w, "cannot overwrite built-in template", http.StatusForbidden)
		return
	}

	// SEC-06: Verify resolved path is within TemplatesDir
	tplDir := filepath.Join(s.manager.TemplatesDir, name)
	absTemplatesDir, _ := filepath.Abs(s.manager.TemplatesDir)
	absTplDir, _ := filepath.Abs(tplDir)
	if !strings.HasPrefix(absTplDir, absTemplatesDir+string(filepath.Separator)) {
		writeJSONError(w, "template directory escapes base", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(tplDir, 0700); err != nil {
		log.Printf("[templates] import: mkdir failed: %v", err)
		writeJSONError(w, "failed to create template directory", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(filepath.Join(tplDir, "manifest.json"), manifestData, 0644); err != nil {
		log.Printf("[templates] import: write manifest failed: %v", err)
		writeJSONError(w, "failed to write manifest", http.StatusInternalServerError)
		return
	}

	binaryName := fmt.Sprintf("presto-template-%s", name)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	rc, err := binaryFile.Open()
	if err != nil {
		writeJSONError(w, "failed to read binary from ZIP", http.StatusInternalServerError)
		return
	}
	// SEC-13: Limit binary size to 100MB
	binData, err := io.ReadAll(io.LimitReader(rc, 100<<20))
	rc.Close()
	if err != nil {
		writeJSONError(w, "failed to read binary", http.StatusInternalServerError)
		return
	}

	binaryPath := filepath.Join(tplDir, binaryName)
	if err := os.WriteFile(binaryPath, binData, 0755); err != nil {
		log.Printf("[templates] import: write binary failed: %v", err)
		writeJSONError(w, "failed to write binary", http.StatusInternalServerError)
		return
	}

	log.Printf("[templates] imported template %q from ZIP", name)

	type importResult struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Author      string   `json:"author"`
		Builtin     bool     `json:"builtin"`
		Keywords    []string `json:"keywords"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(importResult{
		Name:        manifest.Name,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Builtin:     false,
		Keywords:    manifest.Keywords,
	})
}
