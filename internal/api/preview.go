package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mrered/presto/internal/preview"
)

type previewUpdateRequest struct {
	Markdown    string `json:"markdown"`
	TemplateID  string `json:"templateId"`
	WorkDir     string `json:"workDir,omitempty"`
	DocumentKey string `json:"documentKey,omitempty"`
}

type previewUpdateResponse struct {
	Events   []preview.Event `json:"events"`
	SVGPages []string        `json:"svgPages,omitempty"`
}

func (s *Server) handlePreviewUpdate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var req previewUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[preview-update] invalid request: %v", err)
		writeJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := validateWorkDir(req.WorkDir); err != nil {
		log.Printf("[preview-update] invalid workDir %q: %v", req.WorkDir, err)
		writeJSONError(w, "invalid work directory", http.StatusBadRequest)
		return
	}

	identity := preview.DocumentIdentity{
		TemplateID:  req.TemplateID,
		WorkDir:     req.WorkDir,
		DocumentKey: req.DocumentKey,
	}
	result := s.previewService.BeginUpdate(identity)
	events := append([]preview.Event{}, result.Events...)

	tpl, err := s.manager.Get(req.TemplateID)
	if err != nil {
		log.Printf("[preview-update] template not found: %s: %v", req.TemplateID, err)
		events = append(events, s.previewService.ConversionFailed(result.Version, err))
		writePreviewUpdateResponse(w, previewUpdateResponse{Events: events})
		return
	}

	typstOutput, err := s.manager.Executor(tpl).Convert(req.Markdown)
	if err != nil {
		log.Printf("[preview-update] conversion failed: %v", err)
		events = append(events, s.previewService.ConversionFailed(result.Version, err))
		writePreviewUpdateResponse(w, previewUpdateResponse{Events: events})
		return
	}

	svgPages, err := s.compiler.CompileToSVG(typstOutput, req.WorkDir)
	if err != nil {
		log.Printf("[preview-update] fallback compile failed: %v", err)
		events = append(events, s.previewService.FallbackFailed(result.Version, err))
		writePreviewUpdateResponse(w, previewUpdateResponse{Events: events})
		return
	}

	if event, ok := s.previewService.ApplyFallback(result.Version, preview.PagesFromSVG(svgPages)); ok {
		events = append(events, event)
	}
	writePreviewUpdateResponse(w, previewUpdateResponse{
		Events:   events,
		SVGPages: svgPages,
	})
}

func (s *Server) handlePreviewEvents(w http.ResponseWriter, r *http.Request) {
	// server embedded Tinymist proxy deferred; preview events endpoint preserves origin/WebSocket boundary
	// TODO: server Tinymist embedded renderer deferred; this endpoint preserves WebSocket/origin boundary
	writeJSONError(w, "server Tinymist embedded renderer deferred", http.StatusNotImplemented)
}

func writePreviewUpdateResponse(w http.ResponseWriter, resp previewUpdateResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
