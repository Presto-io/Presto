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
	"regexp"
	"strings"

	"github.com/mrered/presto/internal/template"
)

// BatchImportResult is the result of processing a batch ZIP file.
// Exported so the desktop Wails binding can return it directly.
type BatchImportResult struct {
	Templates     []TemplateImportStatus `json:"templates"`
	MarkdownFiles []MarkdownFileEntry    `json:"markdownFiles"`
	WorkDir       string                 `json:"workDir,omitempty"`
}

// TemplateImportStatus describes the outcome of importing a single template.
type TemplateImportStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"` // "installed", "overwritten", "skipped"
}

// MarkdownFileEntry describes a markdown file extracted from a ZIP.
type MarkdownFileEntry struct {
	Name             string `json:"name"`
	Content          string `json:"content"`
	DetectedTemplate string `json:"detectedTemplate,omitempty"`
	WorkDir          string `json:"workDir,omitempty"`
}

// Markdown file extensions for batch import.
var markdownExts = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
}

// ProcessBatchZip processes ZIP data: extracts templates and markdown files.
// Shared by the HTTP handler and the desktop Wails binding.
func ProcessBatchZip(zipData []byte, mgr *template.Manager, registry *template.RegistryCache) (*BatchImportResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("invalid ZIP file: %w", err)
	}

	// SEC-30: Reject path traversal in ZIP entries
	for _, f := range zr.File {
		if strings.Contains(f.Name, "..") {
			return nil, fmt.Errorf("ZIP contains path traversal")
		}
	}

	// Identify template directories (those containing manifest.json)
	templateRoots := findTemplateRoots(zr)
	templateRootSet := make(map[string]bool)
	for _, root := range templateRoots {
		templateRootSet[root] = true
	}

	// Create a temp directory to extract non-template files (markdown, images, etc.)
	workDir, err := os.MkdirTemp("", "presto-batch-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	// Extract non-template files to workDir, preserving directory structure
	var mdFiles []string // relative paths of markdown files
	for _, f := range zr.File {
		if skipZipEntry(f) {
			continue
		}
		// Skip files that belong to template directories
		if isInsideTemplateRoot(f.Name, templateRootSet) {
			continue
		}

		// Extract to workDir
		relPath := f.Name
		destPath := filepath.Join(workDir, filepath.FromSlash(relPath))

		// SEC-06: Verify resolved path is within workDir
		absWorkDir, _ := filepath.Abs(workDir)
		absDest, _ := filepath.Abs(destPath)
		if !strings.HasPrefix(absDest, absWorkDir+string(filepath.Separator)) {
			continue // skip files that would escape workDir
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// SEC-13: Limit individual file size (10MB for non-binary files)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		fileData, err := io.ReadAll(io.LimitReader(rc, 10<<20))
		rc.Close()
		if err != nil {
			continue
		}

		if err := os.WriteFile(destPath, fileData, 0644); err != nil {
			continue
		}

		// Track markdown files
		ext := strings.ToLower(filepath.Ext(relPath))
		if markdownExts[ext] {
			mdFiles = append(mdFiles, relPath)
		}
	}

	// Import templates (always overwrite if same name exists)
	importedTemplates := make([]TemplateImportStatus, 0)
	for _, root := range templateRoots {
		manifest, err := readManifestFromZip(zr, root)
		if err != nil {
			log.Printf("[batch] import: skipping template at %q: %v", root, err)
			continue
		}

		name := manifest.Name
		status := "installed"

		if mgr.Exists(name) {
			// Always overwrite for batch import
			if err := mgr.Uninstall(name); err != nil {
				log.Printf("[batch] import: failed to uninstall %q: %v", name, err)
				continue
			}
			status = "overwritten"
		}

		result, err := importTemplateFromZipDir(zr, root, name, mgr, registry)
		if err != nil {
			log.Printf("[batch] import: failed to import template %q: %v", name, err)
			continue
		}

		importedTemplates = append(importedTemplates, TemplateImportStatus{
			Name:        result.Name,
			DisplayName: result.DisplayName,
			Status:      status,
		})
	}

	// Read markdown files and extract template field from frontmatter
	markdownEntries := make([]MarkdownFileEntry, 0, len(mdFiles))
	for _, relPath := range mdFiles {
		absPath := filepath.Join(workDir, filepath.FromSlash(relPath))
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		entry := MarkdownFileEntry{
			Name:    path.Base(relPath),
			Content: string(content),
		}

		// Per-file workDir: resolve image paths relative to the markdown file's directory
		fileDir := path.Dir(relPath)
		if fileDir != "." {
			entry.WorkDir = filepath.Join(workDir, filepath.FromSlash(fileDir))
		}

		entry.DetectedTemplate = extractFrontmatterTemplate(string(content))

		markdownEntries = append(markdownEntries, entry)
	}

	result := &BatchImportResult{
		Templates:     importedTemplates,
		MarkdownFiles: markdownEntries,
	}

	// Only include workDir if there are non-markdown files (images, etc.)
	// that the Typst compiler might need to reference
	if hasNonMarkdownFiles(workDir, mdFiles) {
		result.WorkDir = workDir
	} else {
		// Clean up if no images/assets to reference
		os.RemoveAll(workDir)
	}

	log.Printf("[batch] imported %d template(s), %d markdown file(s) from ZIP (workDir=%s)",
		len(importedTemplates), len(markdownEntries), result.WorkDir)

	return result, nil
}

func (s *Server) handleBatchImportZip(w http.ResponseWriter, r *http.Request) {
	// SEC-11: Limit request body
	r.Body = http.MaxBytesReader(w, r.Body, maxZIPUploadSize)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("[batch] import: parse form failed: %v", err)
		writeJSONError(w, "invalid request or file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[batch] import: no file in request: %v", err)
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
		log.Printf("[batch] import: read file failed: %v", err)
		writeJSONError(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}

	result, err := ProcessBatchZip(data, s.manager, s.registry)
	if err != nil {
		log.Printf("[batch] import: %v", err)
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// isInsideTemplateRoot checks if a file path belongs to any template root directory.
func isInsideTemplateRoot(filePath string, roots map[string]bool) bool {
	dir := path.Dir(filePath)
	for dir != "." && dir != "" {
		if roots[dir] {
			return true
		}
		dir = path.Dir(dir)
	}
	// Check root-level template (prefix = "")
	if roots[""] {
		// If root-level has manifest.json, all files at root could be template files.
		// But we should be more specific — only skip manifest.json and binary at root.
		return path.Base(filePath) == "manifest.json"
	}
	return false
}

// hasNonMarkdownFiles checks if the workDir contains files other than the markdown files.
func hasNonMarkdownFiles(workDir string, mdRelPaths []string) bool {
	mdSet := make(map[string]bool)
	for _, p := range mdRelPaths {
		mdSet[filepath.FromSlash(p)] = true
	}

	hasOther := false
	filepath.Walk(workDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workDir, p)
		if err != nil {
			return nil
		}
		if !mdSet[rel] {
			hasOther = true
			return filepath.SkipAll
		}
		return nil
	})
	return hasOther
}

// extractFrontmatterTemplate extracts the `template` field from YAML frontmatter.
var frontmatterTemplateRe = regexp.MustCompile(`(?m)^template\s*:\s*(.+)$`)

func extractFrontmatterTemplate(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if !strings.HasPrefix(trimmed, "---") {
		return ""
	}
	endIdx := strings.Index(trimmed[3:], "\n---")
	if endIdx == -1 {
		return ""
	}
	frontmatter := trimmed[3 : 3+endIdx]
	match := frontmatterTemplateRe.FindStringSubmatch(frontmatter)
	if len(match) < 2 {
		return ""
	}
	value := strings.TrimSpace(match[1])
	// Strip quotes
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	// Strip inline comments
	if idx := strings.Index(value, " #"); idx > 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}
