package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxRequestBody = 10 << 20 // 10MB (SEC-11)

type convertRequest struct {
	Markdown   string `json:"markdown"`
	TemplateID string `json:"templateId"`
	WorkDir    string `json:"workDir,omitempty"`
}

type convertResponse struct {
	Typst string `json:"typst"`
}

// validateWorkDir validates the work directory parameter (SEC-03).
func validateWorkDir(workDir string) error {
	if workDir == "" {
		return nil
	}
	if !filepath.IsAbs(workDir) {
		return fmt.Errorf("workDir must be an absolute path")
	}
	clean := filepath.Clean(workDir)
	if strings.Contains(clean, "..") {
		return fmt.Errorf("workDir contains path traversal")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return fmt.Errorf("workDir does not exist")
	}
	if !info.IsDir() {
		return fmt.Errorf("workDir is not a directory")
	}
	return nil
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[convert] invalid request: %v", err)
		writeJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("[convert] template=%s markdown_len=%d", req.TemplateID, len(req.Markdown))

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		log.Printf("[convert] template not found: %s: %v", req.TemplateID, err)
		writeJSONError(w, "template not found", http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		log.Printf("[convert] conversion failed: %v", err)
		writeJSONError(w, "conversion failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[convert] success, typst_len=%d", len(typstOutput))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertResponse{Typst: typstOutput})
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[compile] read body failed: %v", err)
		writeJSONError(w, "request too large or read failed", http.StatusBadRequest)
		return
	}

	workDir := r.Header.Get("X-Work-Dir")
	if workDir == "" {
		workDir = r.URL.Query().Get("workDir")
	}
	if err := validateWorkDir(workDir); err != nil {
		log.Printf("[compile] invalid workDir %q: %v", workDir, err)
		writeJSONError(w, "invalid work directory", http.StatusBadRequest)
		return
	}
	log.Printf("[compile] typst_len=%d workDir=%s", len(body), workDir)

	pdf, err := s.compiler.CompileString(string(body), workDir)
	if err != nil {
		log.Printf("[compile] compile failed: %v", err)
		writeJSONError(w, "compile failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[compile] success, pdf_len=%d", len(pdf))
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(pdf)
}

func (s *Server) handleCompileSVG(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[compile-svg] read body failed: %v", err)
		writeJSONError(w, "request too large or read failed", http.StatusBadRequest)
		return
	}

	workDir := r.Header.Get("X-Work-Dir")
	if workDir == "" {
		workDir = r.URL.Query().Get("workDir")
	}
	if err := validateWorkDir(workDir); err != nil {
		log.Printf("[compile-svg] invalid workDir %q: %v", workDir, err)
		writeJSONError(w, "invalid work directory", http.StatusBadRequest)
		return
	}
	log.Printf("[compile-svg] typst_len=%d workDir=%s", len(body), workDir)

	pages, err := s.compiler.CompileToSVG(string(body), workDir)
	if err != nil {
		log.Printf("[compile-svg] compile failed: %v", err)
		writeJSONError(w, "compile failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[compile-svg] success, pages=%d", len(pages))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"pages": pages})
}

func (s *Server) handleConvertAndCompile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[convert-and-compile] invalid request: %v", err)
		writeJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := validateWorkDir(req.WorkDir); err != nil {
		log.Printf("[convert-and-compile] invalid workDir %q: %v", req.WorkDir, err)
		writeJSONError(w, "invalid work directory", http.StatusBadRequest)
		return
	}

	log.Printf("[convert-and-compile] template=%s markdown_len=%d", req.TemplateID, len(req.Markdown))

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		log.Printf("[convert-and-compile] template not found: %s: %v", req.TemplateID, err)
		writeJSONError(w, "template not found", http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		log.Printf("[convert-and-compile] conversion failed: %v", err)
		writeJSONError(w, "conversion failed", http.StatusInternalServerError)
		return
	}

	pdf, err := s.compiler.CompileString(typstOutput, req.WorkDir)
	if err != nil {
		log.Printf("[convert-and-compile] compile failed: %v", err)
		writeJSONError(w, "compile failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[convert-and-compile] success, pdf_len=%d", len(pdf))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=output.pdf")
	w.Write(pdf)
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, "not implemented", http.StatusNotImplemented)
}
