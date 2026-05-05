package api

import (
	"encoding/json"
	"errors"
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

// InstallErrorResponse provides structured error responses for install operations
type InstallErrorResponse struct {
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
}

// writeInstallError writes a structured error response for template installation
func writeInstallError(w http.ResponseWriter, errType, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(InstallErrorResponse{
		ErrorType: errType,
		Message:   message,
	})
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.manager.List()
	if err != nil {
		log.Printf("[templates] list failed: %v", err)
		writeJSONError(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	log.Printf("[templates] found %d templates", len(templates))

	type missingFont struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		URL         string `json:"url"`
	}

	type templateInfo struct {
		Name         string        `json:"name"`
		DisplayName  string        `json:"displayName"`
		Description  string        `json:"description"`
		Version      string        `json:"version"`
		Author       string        `json:"author"`
		Builtin      bool          `json:"builtin"`
		Keywords     []string      `json:"keywords"`
		MissingFonts []missingFont `json:"missingFonts,omitempty"`
	}

	result := make([]templateInfo, 0, len(templates))
	for _, t := range templates {
		info := templateInfo{
			Name:        t.Manifest.Name,
			DisplayName: t.Manifest.DisplayName,
			Description: t.Manifest.Description,
			Version:     t.Manifest.Version,
			Author:      t.Manifest.Author,
			Keywords:    t.Manifest.Keywords,
		}
		for _, f := range t.Manifest.RequiredFonts {
			if !s.availableFonts[f.Name] {
				info.MissingFonts = append(info.MissingFonts, missingFont{
					Name:        f.Name,
					DisplayName: f.DisplayName,
					URL:         f.URL,
				})
			}
		}
		result = append(result, info)
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

	// SEC-39: Only accept owner/repo, not client-provided URLs
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

	// SEC-39: Server-side registry lookup — never trust client-provided URLs
	var opts *template.InstallOpts
	if s.registry != nil {
		ownerRepo := req.Owner + "/" + req.Repo
		if entry := s.registry.LookupByRepo(ownerRepo); entry != nil {
			platform := template.Platform()
			if platformOpts, ok := entry.InstallOptsForPlatform(platform); ok {
				opts = platformOpts
				log.Printf("[templates] registry lookup for %s: trust=%s, platform=%s", ownerRepo, entry.Trust, platform)
			}
		}
	}

	if err := s.manager.Install(req.Owner, req.Repo, opts); err != nil {
		log.Printf("[templates] install %s/%s failed: %v", req.Owner, req.Repo, err)

		// Classify error and return structured response
		var installErr *template.InstallError
		if errors.As(err, &installErr) {
			switch installErr.Type {
			case template.ErrNetwork:
				writeInstallError(w, "network_error", "网络连接失败，请检查网络后重试", http.StatusServiceUnavailable)
			case template.ErrNotFound:
				writeInstallError(w, "not_found", "模板不存在", http.StatusNotFound)
			case template.ErrChecksumMismatch:
				writeInstallError(w, "checksum_mismatch", "文件校验失败，可能已损坏，请重试", http.StatusBadRequest)
			default:
				writeInstallError(w, "server_error", "服务器暂时不可用，请稍后重试", http.StatusInternalServerError)
			}
		} else {
			// Unknown error type - treat as server error
			writeInstallError(w, "server_error", "服务器暂时不可用，请稍后重试", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"installed"}`))
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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
		writeJSONError(w, "rename failed", http.StatusBadRequest) // SEC-35
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
