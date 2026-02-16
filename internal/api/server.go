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

	if staticDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	return corsMiddleware(s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Stub handlers - implemented in templates.go
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
func (s *Server) handleDiscoverTemplates(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
func (s *Server) handleInstallTemplate(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
func (s *Server) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
}
