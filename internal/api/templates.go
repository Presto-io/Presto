package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/mrered/presto/internal/template"
)

// isValidGitHubName validates GitHub owner/repo name format (SEC-17).
var validGitHubNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func isValidGitHubName(name string) bool {
	return len(name) > 0 && len(name) <= 100 && validGitHubNameRe.MatchString(name)
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.manager.List()
	if err != nil {
		log.Printf("[templates] list failed: %v", err)
		writeJSONError(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	log.Printf("[templates] found %d templates", len(templates))

	type templateInfo struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Author      string   `json:"author"`
		Builtin     bool     `json:"builtin"`
		Keywords    []string `json:"keywords"`
	}

	var result []templateInfo
	for _, t := range templates {
		result = append(result, templateInfo{
			Name:        t.Manifest.Name,
			DisplayName: t.Manifest.DisplayName,
			Description: t.Manifest.Description,
			Version:     t.Manifest.Version,
			Author:      t.Manifest.Author,
			Builtin:     template.IsOfficial(t.Manifest.Name),
			Keywords:    t.Manifest.Keywords,
		})
	}

	if result == nil {
		result = []templateInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDiscoverTemplates(w http.ResponseWriter, r *http.Request) {
	repos, err := template.DiscoverTemplates()
	if err != nil {
		log.Printf("[templates] discover failed: %v", err)
		writeJSONError(w, "discovery failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

func (s *Server) handleInstallTemplate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	id := r.PathValue("id")

	var req struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Owner == "" {
		parts := strings.SplitN(id, "/", 2)
		if len(parts) == 2 {
			req.Owner = parts[0]
			req.Repo = parts[1]
		} else {
			writeJSONError(w, "invalid request", http.StatusBadRequest)
			return
		}
	}

	// SEC-17: Validate owner/repo format
	if !isValidGitHubName(req.Owner) || !isValidGitHubName(req.Repo) {
		writeJSONError(w, "invalid owner or repo name", http.StatusBadRequest)
		return
	}

	if err := s.manager.Install(req.Owner, req.Repo); err != nil {
		log.Printf("[templates] install %s/%s failed: %v", req.Owner, req.Repo, err)
		writeJSONError(w, "install failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"installed"}`))
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if template.IsOfficial(id) {
		writeJSONError(w, "cannot delete built-in template", http.StatusForbidden)
		return
	}
	if err := s.manager.Uninstall(id); err != nil {
		log.Printf("[templates] delete %s failed: %v", id, err)
		writeJSONError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"deleted"}`))
}

func (s *Server) handleRenameTemplate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	id := r.PathValue("id")

	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DisplayName == "" {
		writeJSONError(w, "invalid request: displayName is required", http.StatusBadRequest)
		return
	}

	if err := s.manager.UpdateDisplayName(id, req.DisplayName); err != nil {
		log.Printf("[templates] update displayName %s → %q failed: %v", id, req.DisplayName, err)
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[templates] updated template %s displayName → %q", id, req.DisplayName)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "renamed", "displayName": req.DisplayName})
}

func (s *Server) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tpl, err := s.manager.Get(id)
	if err != nil {
		writeJSONError(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tpl.Manifest)
}

func (s *Server) handleGetExample(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tpl, err := s.manager.Get(id)
	if err != nil {
		writeJSONError(w, "not found", http.StatusNotFound)
		return
	}

	exec := s.manager.Executor(tpl)
	example, err := exec.GetExample()
	if err != nil {
		// Template doesn't support --example, return empty
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"example":""}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"example": example})
}
