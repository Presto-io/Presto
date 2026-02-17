package api

import (
	"log"
	"net/http"

	"github.com/mrered/presto/internal/template"
	"github.com/mrered/presto/internal/typst"
)

type Server struct {
	mux      *http.ServeMux
	manager  *template.Manager
	compiler *typst.Compiler
}

func NewServer(templatesDir, staticDir, typstBin string) http.Handler {
	compiler := typst.NewCompilerWithRoot("/")
	compiler.BinPath = typstBin

	s := &Server{
		mux:      http.NewServeMux(),
		manager:  template.NewManager(templatesDir),
		compiler: compiler,
	}

	log.Printf("[presto] starting server, templates=%s static=%s typst=%s", templatesDir, staticDir, typstBin)

	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/convert", s.handleConvert)
	s.mux.HandleFunc("POST /api/compile", s.handleCompile)
	s.mux.HandleFunc("POST /api/compile-svg", s.handleCompileSVG)
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

	return loggingMiddleware(corsMiddleware(s.mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

