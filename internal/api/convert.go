package api

import (
	"encoding/json"
	"io"
	"log"
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
		log.Printf("[convert] invalid request: %v", err)
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[convert] template=%s markdown_len=%d", req.TemplateID, len(req.Markdown))

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		log.Printf("[convert] template not found: %s: %v", req.TemplateID, err)
		http.Error(w, `{"error":"template not found"}`, http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		log.Printf("[convert] conversion failed: %v", err)
		http.Error(w, `{"error":"conversion failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[convert] success, typst_len=%d", len(typstOutput))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertResponse{Typst: typstOutput})
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[compile] read body failed: %v", err)
		http.Error(w, `{"error":"read failed"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[compile] typst_len=%d", len(body))

	pdf, err := s.compiler.CompileString(string(body))
	if err != nil {
		log.Printf("[compile] compile failed: %v", err)
		http.Error(w, `{"error":"compile failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[compile] success, pdf_len=%d", len(pdf))
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(pdf)
}

func (s *Server) handleConvertAndCompile(w http.ResponseWriter, r *http.Request) {
	var req convertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[convert-and-compile] invalid request: %v", err)
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[convert-and-compile] template=%s markdown_len=%d", req.TemplateID, len(req.Markdown))

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		log.Printf("[convert-and-compile] template not found: %s: %v", req.TemplateID, err)
		http.Error(w, `{"error":"template not found"}`, http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	typstOutput, err := exec.Convert(req.Markdown)
	if err != nil {
		log.Printf("[convert-and-compile] conversion failed: %v", err)
		http.Error(w, `{"error":"conversion failed"}`, http.StatusInternalServerError)
		return
	}

	pdf, err := s.compiler.CompileString(typstOutput)
	if err != nil {
		log.Printf("[convert-and-compile] compile failed: %v", err)
		http.Error(w, `{"error":"compile failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[convert-and-compile] success, pdf_len=%d", len(pdf))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=output.pdf")
	w.Write(pdf)
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
