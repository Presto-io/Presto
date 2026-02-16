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
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
