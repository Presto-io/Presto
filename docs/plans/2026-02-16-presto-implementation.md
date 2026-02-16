# Presto Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Markdown → Typst → PDF conversion platform with pluggable template ecosystem.

**Architecture:** Go API server manages template binaries (stdin/stdout protocol) and typst compilation. Svelte frontend provides editor, preview, and template management. Docker for deployment.

**Tech Stack:** Go 1.24, Svelte 5 + SvelteKit, CodeMirror 6, typst.ts, Docker, goreleaser

---

## Phase 1: Backend Foundation

### Task 1: Project Setup

**Files:**
- Create: `go.mod`
- Create: `cmd/presto-server/main.go`
- Create: `.gitignore`

**Step 1: Initialize git and Go module**

```bash
cd /Users/mrered/Developer/Code/Gopst
git init
go mod init github.com/mrered/presto
```

**Step 2: Create .gitignore**

```gitignore
# Binaries
/bin/
*.exe
*.dll
*.so
*.dylib

# Go
/vendor/

# Frontend
/frontend/node_modules/
/frontend/.svelte-kit/
/frontend/build/

# OS
.DS_Store
Thumbs.db

# Presto data
.presto/
cache/
```

**Step 3: Create directory structure**

```bash
mkdir -p cmd/presto-server internal/api internal/template internal/typst frontend
```

**Step 4: Create minimal server entry point**

`cmd/presto-server/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mrered/presto/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := api.NewServer()
	fmt.Printf("Presto server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
```

**Step 5: Create stub API server**

`internal/api/server.go`:
```go
package api

import "net/http"

func NewServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}
```

**Step 6: Commit**

```bash
git add go.mod cmd/ internal/ .gitignore
git commit -m "feat: project setup with minimal API server"
```

---

### Task 2: Template Manifest Types

**Files:**
- Create: `internal/template/manifest.go`
- Create: `internal/template/manifest_test.go`

**Step 1: Write the test**

`internal/template/manifest_test.go`:
```go
package template

import (
	"testing"
)

func TestParseManifest(t *testing.T) {
	data := []byte(`{
		"name": "gongwen",
		"displayName": "中国党政机关公文格式",
		"version": "1.0.0",
		"author": "mrered"
	}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "gongwen" {
		t.Errorf("got name %q, want %q", m.Name, "gongwen")
	}
	if m.DisplayName != "中国党政机关公文格式" {
		t.Errorf("got displayName %q, want %q", m.DisplayName, "中国党政机关公文格式")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/template/ -v`
Expected: FAIL — `ParseManifest` not defined

**Step 3: Implement manifest types**

`internal/template/manifest.go`:
```go
package template

import "encoding/json"

type FieldSchema struct {
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
	Format  string `json:"format,omitempty"`
}

type Manifest struct {
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"displayName"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	Author            string                 `json:"author"`
	License           string                 `json:"license"`
	MinPrestoVersion  string                 `json:"minPrestoVersion"`
	FrontmatterSchema map[string]FieldSchema `json:"frontmatterSchema"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/template/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/template/
git commit -m "feat: template manifest types and parsing"
```

---

### Task 3: Template Executor

**Files:**
- Create: `internal/template/executor.go`
- Create: `internal/template/executor_test.go`

**Step 1: Write the test**

`internal/template/executor_test.go`:
```go
package template

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 创建一个假的模板二进制用于测试（echo stdin to stdout）
func createMockTemplate(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "mock.go")
	bin := filepath.Join(dir, "mock-template")

	code := `package main
import (
	"fmt"
	"io"
	"os"
)
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--manifest" {
		fmt.Print(` + "`" + `{"name":"mock","version":"0.1.0"}` + "`" + `)
		return
	}
	data, _ := io.ReadAll(os.Stdin)
	fmt.Printf("// converted\n%s", data)
}
`
	os.WriteFile(src, []byte(code), 0644)
	cmd := exec.Command("go", "build", "-o", bin, src)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build mock template: %v", err)
	}
	return bin
}

func TestExecute(t *testing.T) {
	dir := t.TempDir()
	bin := createMockTemplate(t, dir)

	exec := NewExecutor(bin)
	result, err := exec.Convert("# Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "// converted\n# Hello" {
		t.Errorf("got %q, want %q", result, "// converted\n# Hello")
	}
}

func TestExecuteManifest(t *testing.T) {
	dir := t.TempDir()
	bin := createMockTemplate(t, dir)

	exec := NewExecutor(bin)
	data, err := exec.GetManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if m.Name != "mock" {
		t.Errorf("got name %q, want %q", m.Name, "mock")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/template/ -run TestExecute -v`
Expected: FAIL — `NewExecutor` not defined

**Step 3: Implement executor**

`internal/template/executor.go`:
```go
package template

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Executor struct {
	BinaryPath string
}

func NewExecutor(binaryPath string) *Executor {
	return &Executor{BinaryPath: binaryPath}
}

func (e *Executor) Convert(markdown string) (string, error) {
	cmd := exec.Command(e.BinaryPath)
	cmd.Stdin = strings.NewReader(markdown)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("template execution failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (e *Executor) GetManifest() ([]byte, error) {
	cmd := exec.Command(e.BinaryPath, "--manifest")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("manifest retrieval failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/template/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/template/
git commit -m "feat: template executor with stdin/stdout pipe"
```

---

### Task 4: Template Manager

**Files:**
- Create: `internal/template/manager.go`
- Create: `internal/template/manager_test.go`

**Step 1: Write the test**

`internal/template/manager_test.go`:
```go
package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerListTemplates(t *testing.T) {
	dir := t.TempDir()

	// 创建一个假模板目录
	tplDir := filepath.Join(dir, "templates", "mock")
	os.MkdirAll(tplDir, 0755)

	manifest := `{"name":"mock","displayName":"Mock Template","version":"0.1.0","author":"test"}`
	os.WriteFile(filepath.Join(tplDir, "manifest.json"), []byte(manifest), 0644)

	// 创建假二进制
	bin := createMockTemplate(t, t.TempDir())
	copyFile(t, bin, filepath.Join(tplDir, "presto-template-mock"))

	mgr := NewManager(filepath.Join(dir, "templates"))
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

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(dst, data, 0755)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/template/ -run TestManagerList -v`
Expected: FAIL

**Step 3: Implement manager**

`internal/template/manager.go`:
```go
package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type InstalledTemplate struct {
	Manifest   *Manifest
	BinaryPath string
	Dir        string
}

type Manager struct {
	TemplatesDir string
}

func NewManager(templatesDir string) *Manager {
	return &Manager{TemplatesDir: templatesDir}
}

func (m *Manager) List() ([]InstalledTemplate, error) {
	entries, err := os.ReadDir(m.TemplatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var templates []InstalledTemplate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tplDir := filepath.Join(m.TemplatesDir, entry.Name())
		manifestPath := filepath.Join(tplDir, "manifest.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // skip directories without manifest
		}

		manifest, err := ParseManifest(data)
		if err != nil {
			continue
		}

		binaryName := fmt.Sprintf("presto-template-%s", manifest.Name)
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(tplDir, binaryName)

		if _, err := os.Stat(binaryPath); err != nil {
			continue // skip if binary missing
		}

		templates = append(templates, InstalledTemplate{
			Manifest:   manifest,
			BinaryPath: binaryPath,
			Dir:        tplDir,
		})
	}
	return templates, nil
}

func (m *Manager) Get(name string) (*InstalledTemplate, error) {
	templates, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		if t.Manifest.Name == name {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}
```

**Step 4: Run tests**

Run: `go test ./internal/template/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/template/
git commit -m "feat: template manager for listing installed templates"
```

---

### Task 5: Typst Compiler Wrapper

**Files:**
- Create: `internal/typst/compiler.go`
- Create: `internal/typst/compiler_test.go`

**Step 1: Write the test**

`internal/typst/compiler_test.go`:
```go
package typst

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCompile(t *testing.T) {
	// 检查 typst 是否安装
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed, skipping")
	}

	dir := t.TempDir()
	typFile := filepath.Join(dir, "test.typ")
	os.WriteFile(typFile, []byte(`= Hello World`), 0644)

	c := NewCompiler()
	pdfPath, err := c.Compile(typFile)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("PDF not created: %v", err)
	}
}

func TestCompileFromString(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed, skipping")
	}

	c := NewCompiler()
	pdf, err := c.CompileString("= Hello World")
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("empty PDF output")
	}
	// PDF magic bytes
	if string(pdf[:5]) != "%PDF-" {
		t.Fatalf("not a valid PDF, got header: %q", string(pdf[:5]))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/typst/ -v`
Expected: FAIL

**Step 3: Implement compiler**

`internal/typst/compiler.go`:
```go
package typst

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(typFile string) (string, error) {
	pdfFile := strings.TrimSuffix(typFile, ".typ") + ".pdf"
	cmd := exec.Command("typst", "compile", typFile, pdfFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("typst compile failed: %w\noutput: %s", err, output)
	}
	return pdfFile, nil
}

func (c *Compiler) CompileString(typstSource string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "presto-compile-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	typFile := filepath.Join(dir, "input.typ")
	if err := os.WriteFile(typFile, []byte(typstSource), 0644); err != nil {
		return nil, err
	}

	pdfFile, err := c.Compile(typFile)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(pdfFile)
}
```

**Step 4: Run tests**

Run: `go test ./internal/typst/ -v`
Expected: PASS (if typst installed) or SKIP

**Step 5: Commit**

```bash
git add internal/typst/
git commit -m "feat: typst compiler wrapper"
```

---

## Phase 2: API Server

### Task 6: HTTP Server with CORS and Static Files

**Files:**
- Modify: `internal/api/server.go`
- Create: `internal/api/middleware.go`

**Step 1: Create CORS middleware**

`internal/api/middleware.go`:
```go
package api

import "net/http"

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

**Step 2: Update server with all route stubs**

`internal/api/server.go`:
```go
package api

import (
	"net/http"

	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
)

type Server struct {
	mux      *http.ServeMux
	manager  *template.Manager
	compiler *typst.Compiler
}

func NewServer(templatesDir, staticDir string) http.Handler {
	s := &Server{
		mux:      http.NewServeMux(),
		manager:  template.NewManager(templatesDir),
		compiler: typst.NewCompiler(),
	}

	// API routes
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/convert", s.handleConvert)
	s.mux.HandleFunc("POST /api/compile", s.handleCompile)
	s.mux.HandleFunc("POST /api/convert-and-compile", s.handleConvertAndCompile)
	s.mux.HandleFunc("POST /api/batch", s.handleBatch)
	s.mux.HandleFunc("GET /api/templates", s.handleListTemplates)
	s.mux.HandleFunc("GET /api/templates/discover", s.handleDiscoverTemplates)
	s.mux.HandleFunc("POST /api/templates/{id}/install", s.handleInstallTemplate)
	s.mux.HandleFunc("DELETE /api/templates/{id}", s.handleDeleteTemplate)
	s.mux.HandleFunc("GET /api/templates/{id}/manifest", s.handleGetManifest)

	// Static files (Svelte build)
	if staticDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	return corsMiddleware(s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
```

**Step 3: Update main.go**

`cmd/presto-server/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mrered/presto/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	home, _ := os.UserHomeDir()
	templatesDir := filepath.Join(home, ".presto", "templates")
	os.MkdirAll(templatesDir, 0755)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/build"
	}

	srv := api.NewServer(templatesDir, staticDir)
	fmt.Printf("Presto server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
```

**Step 4: Commit**

```bash
git add cmd/ internal/api/
git commit -m "feat: HTTP server skeleton with CORS and routing"
```

---

### Task 7: Convert and Compile Endpoints

**Files:**
- Create: `internal/api/convert.go`

**Step 1: Implement handlers**

`internal/api/convert.go`:
```go
package api

import (
	"encoding/json"
	"io"
	"net/http"
)

type convertRequest struct {
	Markdown   string `json:"markdown"`
	TemplateID string `json:"templateId"`
}

type convertResponse struct {
	Typst string `json:"typst"`
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		http.Error(w, `{"error":"conversion failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertResponse{Typst: typstOutput})
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read failed"}`, http.StatusBadRequest)
		return
	}

	pdf, err := s.compiler.CompileString(string(body))
	if err != nil {
		http.Error(w, `{"error":"compile failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Write(pdf)
}

func (s *Server) handleConvertAndCompile(w http.ResponseWriter, r *http.Request) {
	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		http.Error(w, `{"error":"template not found"}`, http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		http.Error(w, `{"error":"conversion failed"}`, http.StatusInternalServerError)
		return
	}

	pdf, err := s.compiler.CompileString(typstOutput)
	if err != nil {
		http.Error(w, `{"error":"compile failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=output.pdf")
	w.Write(pdf)
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	// Phase 5 实现
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
```

**Step 2: Add Executor helper to Manager**

在 `internal/template/manager.go` 末尾追加：
```go
func (m *Manager) Executor(t *InstalledTemplate) *Executor {
	return NewExecutor(t.BinaryPath)
}
```

**Step 3: Commit**

```bash
git add internal/
git commit -m "feat: convert and compile API endpoints"
```

---

### Task 8: Template Management Endpoints

**Files:**
- Create: `internal/api/templates.go`
- Create: `internal/template/github.go`

**Step 1: Implement GitHub discovery**

`internal/template/github.go`:
```go
package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type GitHubRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name string `json:"name"`
}

type GitHubSearchResult struct {
	Items []GitHubRepo `json:"items"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func DiscoverTemplates() ([]GitHubRepo, error) {
	resp, err := http.Get("https://api.github.com/search/repositories?q=topic:presto-template&sort=stars")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GitHubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (m *Manager) Install(owner, repo string) error {
	// 获取最新 release
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	// 查找匹配当前平台的 asset
	pattern := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if contains(asset.Name, pattern) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// 下载并解压
	name := repo // e.g. presto-template-gongwen → extract "gongwen"
	if len(name) > len("presto-template-") {
		name = name[len("presto-template-"):]
	}

	tplDir := filepath.Join(m.TemplatesDir, name)
	os.MkdirAll(tplDir, 0755)

	// 下载 tar.gz 并解压（简化：直接下载二进制）
	binResp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer binResp.Body.Close()

	binPath := filepath.Join(tplDir, repo)
	f, err := os.OpenFile(binPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, binResp.Body)
	return err
}

func (m *Manager) Uninstall(name string) error {
	tplDir := filepath.Join(m.TemplatesDir, name)
	return os.RemoveAll(tplDir)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Implement template API handlers**

`internal/api/templates.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mrered/presto/internal/template"
)

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.manager.List()
	if err != nil {
		http.Error(w, `{"error":"failed to list templates"}`, http.StatusInternalServerError)
		return
	}

	type templateInfo struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Author      string `json:"author"`
	}

	var result []templateInfo
	for _, t := range templates {
		result = append(result, templateInfo{
			Name:        t.Manifest.Name,
			DisplayName: t.Manifest.DisplayName,
			Description: t.Manifest.Description,
			Version:     t.Manifest.Version,
			Author:      t.Manifest.Author,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDiscoverTemplates(w http.ResponseWriter, r *http.Request) {
	repos, err := template.DiscoverTemplates()
	if err != nil {
		http.Error(w, `{"error":"discovery failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

func (s *Server) handleInstallTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 尝试从 id 推断
		parts := strings.SplitN(id, "/", 2)
		if len(parts) == 2 {
			req.Owner = parts[0]
			req.Repo = parts[1]
		} else {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
	}

	if err := s.manager.Install(req.Owner, req.Repo); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"installed"}`))
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Uninstall(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status":"deleted"}`))
}

func (s *Server) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tpl, err := s.manager.Get(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tpl.Manifest)
}
```

**Step 3: Commit**

```bash
git add internal/
git commit -m "feat: template management and GitHub discovery endpoints"
```

---

## Phase 3: Refactor Existing Templates

### Task 9: Refactor Gongwen Template

**Files:**
- Modify: `cmd/gongwen/main.go`
- Create: `cmd/gongwen/manifest.json`

**Step 1: Create manifest.json**

`cmd/gongwen/manifest.json`:
```json
{
  "name": "gongwen",
  "displayName": "中国党政机关公文格式",
  "description": "符合 GB/T 9704-2012 标准的公文排版，支持标题、作者、日期、签名等元素",
  "version": "1.0.0",
  "author": "mrered",
  "license": "MIT",
  "minPrestoVersion": "0.1.0",
  "frontmatterSchema": {
    "title": { "type": "string", "default": "请输入文字" },
    "author": { "type": "string", "default": "请输入文字" },
    "date": { "type": "string", "format": "YYYY-MM-DD" },
    "signature": { "type": "boolean", "default": false }
  }
}
```

**Step 2: Update main.go to support stdin/stdout + --manifest**

在 `cmd/gongwen/main.go` 的 `main()` 函数中，替换现有 CLI 逻辑：

```go
func main() {
	manifestFlag := flag.Bool("manifest", false, "output manifest JSON")
	outputFile := flag.String("o", "", "output .typ file (default: stdout)")
	flag.Parse()

	// --manifest 模式：输出 manifest.json
	if *manifestFlag {
		data, _ := os.ReadFile("manifest.json")
		if len(data) == 0 {
			// fallback: 嵌入的 manifest
			fmt.Print(embeddedManifest)
		} else {
			fmt.Print(string(data))
		}
		return
	}

	// 读取输入：优先文件参数，否则 stdin
	var input []byte
	var err error
	args := flag.Args()
	if len(args) > 0 {
		input, err = os.ReadFile(args[0])
	} else {
		input, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	fm, body := parseFrontMatter(string(input))
	result := convert(fm, body)

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(result), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *outputFile, err)
			os.Exit(1)
		}
	} else {
		fmt.Print(result)
	}
}
```

需要在文件顶部添加 `"io"` 到 import，并添加嵌入的 manifest：

```go
//go:embed manifest.json
var embeddedManifest string
```

**Step 3: 测试 stdin/stdout 模式**

```bash
echo '---
title: 测试文档
author: 张三
date: 2026-01-01
---
## 第一章
这是正文内容。' | go run ./cmd/gongwen/
```

Expected: Typst 源码输出到 stdout

**Step 4: 测试 --manifest 模式**

```bash
go run ./cmd/gongwen/ --manifest
```

Expected: 输出 manifest.json 内容

**Step 5: Commit**

```bash
git add cmd/gongwen/
git commit -m "feat: refactor gongwen template to presto protocol"
```

---

### Task 10: Refactor Jiaoan Template

**Files:**
- Modify: `cmd/jiaoan-shicao/main.go`
- Create: `cmd/jiaoan-shicao/manifest.json`

**Step 1: Create manifest.json**

`cmd/jiaoan-shicao/manifest.json`:
```json
{
  "name": "jiaoan-shicao",
  "displayName": "实操教案格式化",
  "description": "将 Markdown 格式的实操教案转换为标准表格排版",
  "version": "1.0.0",
  "author": "mrered",
  "license": "MIT",
  "minPrestoVersion": "0.1.0"
}
```

**Step 2: Update main.go 支持 stdin 和 --manifest**

与 Task 9 相同模式：添加 `--manifest` flag，支持 stdin 输入，stdout 输出。

**Step 3: Commit**

```bash
git add cmd/jiaoan-shicao/
git commit -m "feat: refactor jiaoan template to presto protocol"
```

---

## Phase 4: Svelte Frontend

### Task 11: SvelteKit Project Setup

**Step 1: Create SvelteKit project**

```bash
cd /Users/mrered/Developer/Code/Gopst
npm create svelte@latest frontend -- --template skeleton --types typescript
cd frontend
npm install
npm install -D @sveltejs/adapter-static
```

**Step 2: Configure adapter-static**

`frontend/svelte.config.js`:
```js
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  kit: {
    adapter: adapter({ pages: 'build', assets: 'build', fallback: 'index.html' })
  },
  preprocess: vitePreprocess()
};
```

**Step 3: Install dependencies**

```bash
cd frontend
npm install codemirror @codemirror/lang-markdown @codemirror/theme-one-dark
npm install @codemirror/view @codemirror/state
```

**Step 4: Commit**

```bash
git add frontend/
git commit -m "feat: SvelteKit project setup with adapter-static"
```

---

### Task 12: API Client and Types

**Files:**
- Create: `frontend/src/lib/api/types.ts`
- Create: `frontend/src/lib/api/client.ts`

**Step 1: Define types**

`frontend/src/lib/api/types.ts`:
```typescript
export interface Template {
  name: string;
  displayName: string;
  description: string;
  version: string;
  author: string;
}

export interface FieldSchema {
  type: string;
  default?: unknown;
  format?: string;
}

export interface Manifest extends Template {
  license: string;
  minPrestoVersion: string;
  frontmatterSchema?: Record<string, FieldSchema>;
}

export interface GitHubRepo {
  full_name: string;
  description: string;
  html_url: string;
  owner: { login: string };
  name: string;
}
```

**Step 2: Implement API client**

`frontend/src/lib/api/client.ts`:
```typescript
import type { Template, Manifest, GitHubRepo } from './types';

const BASE = import.meta.env.VITE_API_URL || '';

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function listTemplates(): Promise<Template[]> {
  return api('/api/templates');
}

export async function discoverTemplates(): Promise<GitHubRepo[]> {
  return api('/api/templates/discover');
}

export async function getManifest(id: string): Promise<Manifest> {
  return api(`/api/templates/${id}/manifest`);
}

export async function convert(markdown: string, templateId: string): Promise<string> {
  const res = await fetch(`${BASE}/api/convert`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ markdown, templateId })
  });
  const data = await res.json();
  return data.typst;
}

export async function convertAndCompile(markdown: string, templateId: string): Promise<Blob> {
  const res = await fetch(`${BASE}/api/convert-and-compile`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ markdown, templateId })
  });
  return res.blob();
}

export async function installTemplate(owner: string, repo: string): Promise<void> {
  await fetch(`${BASE}/api/templates/${owner}%2F${repo}/install`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ owner, repo })
  });
}

export async function deleteTemplate(id: string): Promise<void> {
  await fetch(`${BASE}/api/templates/${id}`, { method: 'DELETE' });
}
```

**Step 3: Commit**

```bash
git add frontend/src/lib/api/
git commit -m "feat: API client and TypeScript types"
```

---

### Task 13: Markdown Editor Component

**Files:**
- Create: `frontend/src/lib/components/Editor.svelte`

**Step 1: Create CodeMirror editor component**

`frontend/src/lib/components/Editor.svelte`:
```svelte
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, basicSetup } from 'codemirror';
  import { markdown } from '@codemirror/lang-markdown';
  import { oneDark } from '@codemirror/theme-one-dark';
  import { EditorState } from '@codemirror/state';

  let { value = $bindable(''), onchange }: {
    value?: string;
    onchange?: (val: string) => void;
  } = $props();

  let container: HTMLDivElement;
  let view: EditorView;

  onMount(() => {
    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          basicSetup,
          markdown(),
          oneDark,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              value = update.state.doc.toString();
              onchange?.(value);
            }
          })
        ]
      }),
      parent: container
    });
  });

  onDestroy(() => view?.destroy());
</script>

<div bind:this={container} class="editor-container"></div>

<style>
  .editor-container {
    height: 100%;
    overflow: auto;
  }
  .editor-container :global(.cm-editor) {
    height: 100%;
  }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/lib/components/
git commit -m "feat: CodeMirror markdown editor component"
```

---

### Task 14: Template Selector Component

**Files:**
- Create: `frontend/src/lib/components/TemplateSelector.svelte`

**Step 1: Create component**

`frontend/src/lib/components/TemplateSelector.svelte`:
```svelte
<script lang="ts">
  import { listTemplates } from '$lib/api/client';
  import type { Template } from '$lib/api/types';

  let { selected = $bindable('') }: { selected?: string } = $props();
  let templates: Template[] = $state([]);

  $effect(() => {
    listTemplates().then(t => {
      templates = t ?? [];
      if (!selected && templates.length > 0) {
        selected = templates[0].name;
      }
    });
  });
</script>

<select bind:value={selected}>
  {#each templates as tpl}
    <option value={tpl.name}>{tpl.displayName || tpl.name}</option>
  {/each}
</select>
```

**Step 2: Commit**

```bash
git add frontend/src/lib/components/
git commit -m "feat: template selector component"
```

---

### Task 15: Preview Component

**Files:**
- Create: `frontend/src/lib/components/Preview.svelte`

**Step 1: Create preview component**

预览使用后端返回的 Typst 源码，通过 typst.ts 渲染为 SVG。初期可先用纯文本预览 Typst 源码，后续接入 typst.ts。

`frontend/src/lib/components/Preview.svelte`:
```svelte
<script lang="ts">
  let { typstSource = '' }: { typstSource?: string } = $props();
</script>

<div class="preview-container">
  {#if typstSource}
    <pre class="typst-source">{typstSource}</pre>
  {:else}
    <p class="placeholder">在左侧编辑 Markdown，选择模板后预览将在此显示</p>
  {/if}
</div>

<style>
  .preview-container {
    height: 100%;
    overflow: auto;
    padding: 1rem;
    background: #1e1e1e;
    color: #d4d4d4;
  }
  .typst-source {
    white-space: pre-wrap;
    font-family: monospace;
    font-size: 0.875rem;
    line-height: 1.5;
  }
  .placeholder {
    color: #666;
    text-align: center;
    margin-top: 2rem;
  }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/lib/components/
git commit -m "feat: typst preview component"
```

---

### Task 16: Editor Page

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Create: `frontend/src/routes/+layout.svelte`

**Step 1: Create layout**

`frontend/src/routes/+layout.svelte`:
```svelte
<script lang="ts">
  let { children } = $props();
</script>

<div class="app">
  <nav>
    <a href="/" class="logo">Presto</a>
    <div class="nav-links">
      <a href="/">编辑器</a>
      <a href="/templates">模板商店</a>
      <a href="/batch">批量转换</a>
      <a href="/settings">设置</a>
    </div>
  </nav>
  <main>
    {@render children()}
  </main>
</div>

<style>
  .app { display: flex; flex-direction: column; height: 100vh; }
  nav { display: flex; align-items: center; padding: 0.5rem 1rem; background: #1a1a2e; color: white; gap: 2rem; }
  .logo { font-weight: bold; font-size: 1.25rem; text-decoration: none; color: white; }
  .nav-links { display: flex; gap: 1rem; }
  .nav-links a { color: #aaa; text-decoration: none; }
  .nav-links a:hover { color: white; }
  main { flex: 1; overflow: hidden; }
</style>
```

**Step 2: Create editor page**

`frontend/src/routes/+page.svelte`:
```svelte
<script lang="ts">
  import Editor from '$lib/components/Editor.svelte';
  import Preview from '$lib/components/Preview.svelte';
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convert, convertAndCompile } from '$lib/api/client';

  let markdown = $state('');
  let typstSource = $state('');
  let selectedTemplate = $state('');
  let debounceTimer: ReturnType<typeof setTimeout>;

  async function handleConvert(md: string) {
    if (!selectedTemplate || !md.trim()) return;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      try {
        typstSource = await convert(md, selectedTemplate);
      } catch (e) {
        console.error('Convert failed:', e);
      }
    }, 500);
  }

  async function handleDownload() {
    if (!selectedTemplate || !markdown.trim()) return;
    try {
      const blob = await convertAndCompile(markdown, selectedTemplate);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'output.pdf';
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error('Download failed:', e);
    }
  }

  function handleUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => { markdown = reader.result as string; };
    reader.readAsText(file);
  }
</script>

<div class="toolbar">
  <TemplateSelector bind:selected={selectedTemplate} />
  <button onclick={handleDownload}>下载 PDF</button>
  <label class="upload-btn">
    上传 MD
    <input type="file" accept=".md,.markdown,.txt" onchange={handleUpload} hidden />
  </label>
</div>

<div class="editor-layout">
  <div class="pane">
    <Editor bind:value={markdown} onchange={handleConvert} />
  </div>
  <div class="pane">
    <Preview {typstSource} />
  </div>
</div>

<style>
  .toolbar { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 1rem; background: #16213e; }
  .toolbar button, .upload-btn { padding: 0.4rem 0.8rem; background: #0f3460; color: white; border: none; border-radius: 4px; cursor: pointer; }
  .toolbar button:hover, .upload-btn:hover { background: #1a5276; }
  .editor-layout { display: flex; flex: 1; height: calc(100vh - 6rem); }
  .pane { flex: 1; overflow: hidden; }
</style>
```

**Step 3: Commit**

```bash
git add frontend/src/routes/
git commit -m "feat: editor page with split pane layout"
```

---

## Phase 5: Additional Pages

### Task 17: Batch Conversion Page

**Files:**
- Create: `frontend/src/routes/batch/+page.svelte`

**Step 1: Create batch conversion page**

`frontend/src/routes/batch/+page.svelte`:
```svelte
<script lang="ts">
  import TemplateSelector from '$lib/components/TemplateSelector.svelte';
  import { convertAndCompile } from '$lib/api/client';

  let selectedTemplate = $state('');
  let files: File[] = $state([]);
  let results: { name: string; blob?: Blob; error?: string }[] = $state([]);
  let processing = $state(false);

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    const dropped = Array.from(e.dataTransfer?.files ?? []).filter(f =>
      f.name.endsWith('.md') || f.name.endsWith('.markdown') || f.name.endsWith('.txt')
    );
    files = [...files, ...dropped];
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    files = [...files, ...Array.from(input.files ?? [])];
  }

  async function convertAll() {
    if (!selectedTemplate || files.length === 0) return;
    processing = true;
    results = [];

    for (const file of files) {
      try {
        const text = await file.text();
        const blob = await convertAndCompile(text, selectedTemplate);
        results = [...results, { name: file.name.replace(/\.\w+$/, '.pdf'), blob }];
      } catch (e) {
        results = [...results, { name: file.name, error: String(e) }];
      }
    }
    processing = false;
  }

  function download(r: { name: string; blob?: Blob }) {
    if (!r.blob) return;
    const url = URL.createObjectURL(r.blob);
    const a = document.createElement('a');
    a.href = url; a.download = r.name; a.click();
    URL.revokeObjectURL(url);
  }
</script>

<div class="batch-page">
  <h2>批量转换</h2>

  <div class="controls">
    <TemplateSelector bind:selected={selectedTemplate} />
    <label class="upload-btn">
      选择文件
      <input type="file" accept=".md,.markdown,.txt" multiple onchange={handleFileInput} hidden />
    </label>
    <button onclick={convertAll} disabled={processing || files.length === 0}>
      {processing ? '转换中...' : `转换 ${files.length} 个文件`}
    </button>
  </div>

  <div class="drop-zone" ondrop={handleDrop} ondragover={(e) => e.preventDefault()}>
    拖拽 Markdown 文件到此处
  </div>

  {#if files.length > 0}
    <ul class="file-list">
      {#each files as file}
        <li>{file.name}</li>
      {/each}
    </ul>
  {/if}

  {#if results.length > 0}
    <h3>转换结果</h3>
    <ul class="results">
      {#each results as r}
        <li>
          {r.name}
          {#if r.blob}
            <button onclick={() => download(r)}>下载</button>
          {:else}
            <span class="error">{r.error}</span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .batch-page { padding: 2rem; max-width: 800px; margin: 0 auto; }
  .controls { display: flex; gap: 0.5rem; margin-bottom: 1rem; align-items: center; }
  .drop-zone { border: 2px dashed #444; padding: 3rem; text-align: center; color: #666; border-radius: 8px; }
  .upload-btn, button { padding: 0.4rem 0.8rem; background: #0f3460; color: white; border: none; border-radius: 4px; cursor: pointer; }
  .file-list { margin-top: 1rem; }
  .error { color: #e74c3c; }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/routes/batch/
git commit -m "feat: batch conversion page"
```

---

### Task 18: Template Store Page

**Files:**
- Create: `frontend/src/routes/templates/+page.svelte`

**Step 1: Create template store page**

`frontend/src/routes/templates/+page.svelte`:
```svelte
<script lang="ts">
  import { listTemplates, discoverTemplates, installTemplate, deleteTemplate } from '$lib/api/client';
  import type { Template, GitHubRepo } from '$lib/api/types';

  let installed: Template[] = $state([]);
  let available: GitHubRepo[] = $state([]);
  let loading = $state(true);

  $effect(() => {
    Promise.all([listTemplates(), discoverTemplates()])
      .then(([inst, avail]) => {
        installed = inst ?? [];
        available = avail ?? [];
        loading = false;
      })
      .catch(() => { loading = false; });
  });

  async function handleInstall(repo: GitHubRepo) {
    await installTemplate(repo.owner.login, repo.name);
    installed = await listTemplates();
  }

  async function handleDelete(name: string) {
    if (!confirm(`确定卸载模板 "${name}"？`)) return;
    await deleteTemplate(name);
    installed = await listTemplates();
  }
</script>

<div class="store-page">
  <h2>已安装模板</h2>
  {#if installed.length === 0}
    <p>暂无已安装模板</p>
  {:else}
    <div class="template-grid">
      {#each installed as tpl}
        <div class="template-card">
          <h3>{tpl.displayName || tpl.name}</h3>
          <p>{tpl.description}</p>
          <div class="card-footer">
            <span>v{tpl.version} · {tpl.author}</span>
            <button class="danger" onclick={() => handleDelete(tpl.name)}>卸载</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <h2>发现更多模板</h2>
  {#if loading}
    <p>加载中...</p>
  {:else if available.length === 0}
    <p>暂无可用模板</p>
  {:else}
    <div class="template-grid">
      {#each available as repo}
        <div class="template-card">
          <h3>{repo.name}</h3>
          <p>{repo.description}</p>
          <div class="card-footer">
            <span>{repo.full_name}</span>
            <button onclick={() => handleInstall(repo)}>安装</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .store-page { padding: 2rem; max-width: 1000px; margin: 0 auto; }
  .template-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
  .template-card { background: #1a1a2e; padding: 1rem; border-radius: 8px; border: 1px solid #333; }
  .template-card h3 { margin: 0 0 0.5rem; }
  .template-card p { color: #aaa; font-size: 0.875rem; }
  .card-footer { display: flex; justify-content: space-between; align-items: center; margin-top: 0.5rem; }
  .card-footer span { font-size: 0.75rem; color: #666; }
  button { padding: 0.3rem 0.6rem; background: #0f3460; color: white; border: none; border-radius: 4px; cursor: pointer; }
  .danger { background: #c0392b; }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/routes/templates/
git commit -m "feat: template store page"
```

---

### Task 19: Settings Page

**Files:**
- Create: `frontend/src/routes/settings/+page.svelte`

**Step 1: Create settings page**

`frontend/src/routes/settings/+page.svelte`:
```svelte
<script lang="ts">
  let communityEnabled = $state(false);
  let showWarning = $state(false);

  function toggleCommunity() {
    if (!communityEnabled) {
      showWarning = true;
    } else {
      communityEnabled = false;
      localStorage.setItem('communityTemplates', 'false');
    }
  }

  function confirmCommunity() {
    communityEnabled = true;
    showWarning = false;
    localStorage.setItem('communityTemplates', 'true');
  }

  $effect(() => {
    communityEnabled = localStorage.getItem('communityTemplates') === 'true';
  });
</script>

<div class="settings-page">
  <h2>设置</h2>

  <section>
    <h3>通用</h3>
    <div class="setting-row">
      <div>
        <strong>启用社区模板</strong>
        <p>允许浏览和安装第三方社区模板</p>
      </div>
      <label class="toggle">
        <input type="checkbox" checked={communityEnabled} onchange={toggleCommunity} />
        <span class="slider"></span>
      </label>
    </div>
  </section>

  <section>
    <h3>模板开发</h3>
    <ul>
      <li><a href="https://github.com/mrered/presto" target="_blank">开发文档</a></li>
      <li>模板协议：可执行文件，stdin 接收 Markdown，stdout 输出 Typst，附带 manifest.json</li>
      <li>支持任意编程语言（Go、Rust、Python、JavaScript 等）</li>
    </ul>
  </section>

  <section>
    <h3>关于 Presto</h3>
    <p>版本：0.1.0</p>
    <p><a href="https://github.com/mrered/presto" target="_blank">GitHub</a></p>
    <p>MIT License</p>
  </section>

  <section>
    <h3>开源协议声明</h3>
    <p>Presto 基于以下开源软件构建，感谢这些项目的贡献者。</p>
    <ul class="licenses">
      <li><strong>Typst</strong> — Apache 2.0</li>
      <li><strong>typst.ts</strong> — Apache 2.0</li>
      <li><strong>Goldmark</strong> — MIT</li>
      <li><strong>CodeMirror</strong> — MIT</li>
      <li><strong>Svelte</strong> — MIT</li>
      <li><strong>Go</strong> — BSD-3-Clause</li>
    </ul>
  </section>

  {#if showWarning}
    <div class="modal-overlay">
      <div class="modal">
        <h3>⚠️ 安全警告</h3>
        <p>社区模板由第三方开发者提供，未经官方审核，可能存在安全风险。请仅安装你信任的模板。</p>
        <div class="modal-actions">
          <button onclick={() => showWarning = false}>取消</button>
          <button class="confirm" onclick={confirmCommunity}>我了解风险，启用</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .settings-page { padding: 2rem; max-width: 700px; margin: 0 auto; }
  section { margin-bottom: 2rem; border-bottom: 1px solid #333; padding-bottom: 1rem; }
  .setting-row { display: flex; justify-content: space-between; align-items: center; }
  .setting-row p { color: #888; font-size: 0.875rem; margin: 0.25rem 0 0; }
  .toggle { position: relative; width: 48px; height: 26px; }
  .toggle input { opacity: 0; width: 0; height: 0; }
  .slider { position: absolute; inset: 0; background: #444; border-radius: 13px; cursor: pointer; transition: 0.3s; }
  .slider::before { content: ''; position: absolute; width: 20px; height: 20px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: 0.3s; }
  .toggle input:checked + .slider { background: #2ecc71; }
  .toggle input:checked + .slider::before { transform: translateX(22px); }
  .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; }
  .modal { background: #1a1a2e; padding: 2rem; border-radius: 12px; max-width: 400px; }
  .modal-actions { display: flex; gap: 0.5rem; margin-top: 1rem; justify-content: flex-end; }
  .modal-actions button { padding: 0.4rem 0.8rem; border: none; border-radius: 4px; cursor: pointer; }
  .confirm { background: #e74c3c; color: white; }
  .licenses li { margin: 0.25rem 0; }
  a { color: #3498db; }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/routes/settings/
git commit -m "feat: settings page with community toggle and licenses"
```

---

## Phase 6: Docker Deployment

### Task 20: Dockerfile

**Files:**
- Create: `Dockerfile`

**Step 1: Create multi-stage Dockerfile**

`Dockerfile`:
```dockerfile
# Stage 1: Build Go API server
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /presto-server ./cmd/presto-server/

# Build official templates
RUN CGO_ENABLED=0 go build -o /presto-template-gongwen ./cmd/gongwen/
RUN CGO_ENABLED=0 go build -o /presto-template-jiaoan ./cmd/jiaoan-shicao/

# Stage 2: Build Svelte frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 3: Final image
FROM alpine:latest

# Install typst
RUN apk add --no-cache curl tar && \
    curl -sSL https://github.com/typst/typst/releases/latest/download/typst-x86_64-unknown-linux-musl.tar.xz | \
    tar -xJ --strip-components=1 -C /usr/local/bin/ && \
    apk del curl tar

# Copy API server
COPY --from=go-builder /presto-server /usr/local/bin/

# Copy official templates
RUN mkdir -p /root/.presto/templates/gongwen /root/.presto/templates/jiaoan-shicao
COPY --from=go-builder /presto-template-gongwen /root/.presto/templates/gongwen/presto-template-gongwen
COPY cmd/gongwen/manifest.json /root/.presto/templates/gongwen/
COPY --from=go-builder /presto-template-jiaoan /root/.presto/templates/jiaoan-shicao/presto-template-jiaoan-shicao
COPY cmd/jiaoan-shicao/manifest.json /root/.presto/templates/jiaoan-shicao/

# Copy frontend build
COPY --from=frontend-builder /app/build /srv/frontend

ENV PORT=8080
ENV STATIC_DIR=/srv/frontend
EXPOSE 8080

CMD ["presto-server"]
```

**Step 2: Commit**

```bash
git add Dockerfile
git commit -m "feat: multi-stage Dockerfile"
```

---

### Task 21: Docker Compose

**Files:**
- Create: `docker-compose.yml`

**Step 1: Create docker-compose.yml**

`docker-compose.yml`:
```yaml
services:
  presto:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - presto-data:/root/.presto
    restart: unless-stopped

volumes:
  presto-data:
```

**Step 2: Test build and run**

```bash
docker compose build
docker compose up -d
curl http://localhost:8080/api/health
```

Expected: `{"status":"ok"}`

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: docker-compose for deployment"
```

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1 | 1-5 | Go 后端基础：模板协议、执行器、管理器、typst 编译器 |
| 2 | 6-8 | API 服务：路由、转换端点、模板管理端点 |
| 3 | 9-10 | 重构现有模板适配新协议 |
| 4 | 11-16 | Svelte 前端：编辑器、预览、模板选择器、主页面 |
| 5 | 17-19 | 批量转换、模板商店、设置页 |
| 6 | 20-21 | Docker 部署 |
