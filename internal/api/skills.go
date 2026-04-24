package api

import (
	"encoding/json"
	"log"
	"net/http"
)

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
			Keywords:    sk.Keywords,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
