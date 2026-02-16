package template

import "encoding/json"

type FieldSchema struct {
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
	Format  string `json:"format,omitempty"`
}

type Manifest struct {
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"displayName"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	Author            string                 `json:"author"`
	License           string                 `json:"license"`
	MinPrestoVersion  string                 `json:"minPrestoVersion"`
	FrontmatterSchema map[string]FieldSchema `json:"frontmatterSchema"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
