package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"regexp"

)

// validSkillNameRe validates skill directory names (presto- prefix + lowercase alphanumeric + hyphens).
var validSkillNameRe = regexp.MustCompile(`^presto-[a-z0-9-]+$`)

// validSourceNames are the known AI tool source names.
var validSourceNames = map[string]bool{
	"codex":     true,
	"claude":    true,
	"workbuddy": true,
	"qclaw":     true,
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := s.skillManager.List()
	if err != nil {
		log.Printf("[skills] list failed: %v", err)
		writeJSONError(w, "failed to list skills", http.StatusInternalServerError)
		return
	}

	log.Printf("[skills] found %d skills", len(skills))

	type skillInfo struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Author      string   `json:"author"`
		Source      string   `json:"source"`
		SourcePath  string   `json:"sourcePath"`
		Keywords    []string `json:"keywords"`
	}

	result := make([]skillInfo, 0, len(skills))
	for _, sk := range skills {
		result = append(result, skillInfo{
			Name:        sk.Name,
			DisplayName: sk.DisplayName,
			Description: sk.Description,
			Version:     sk.Version,
			Author:      sk.Author,
			Source:      sk.Source,
			SourcePath:  sk.SourcePath,
			Keywords:    sk.Keywords,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	name := r.PathValue("name")

	// Validate source name
	if !validSourceNames[source] {
		writeJSONError(w, "invalid source", http.StatusBadRequest)
		return
	}

	// Validate skill name format
	if !validSkillNameRe.MatchString(name) {
		writeJSONError(w, "invalid skill name", http.StatusBadRequest)
		return
	}

	// Find the scan dir matching the source and reconstruct the path
	var sourcePath string
	for _, dir := range s.skillManager.ScanDirs() {
		if dir.Name == source {
			sourcePath = filepath.Join(dir.Path, name)
			break
		}
	}

	if sourcePath == "" {
		writeJSONError(w, "unknown source", http.StatusBadRequest)
		return
	}

	if err := s.skillManager.Delete(sourcePath); err != nil {
		log.Printf("[skills] delete %s/%s failed: %v", source, name, err)
		writeJSONError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[skills] deleted skill %s/%s", source, name)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"deleted"}`))
}
