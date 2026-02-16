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

	if result == nil {
		result = []templateInfo{}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Owner == "" {
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
	w.Header().Set("Content-Type", "application/json")
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
