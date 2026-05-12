package template

import "encoding/json"

type FieldSchema struct {
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
	Format  string `json:"format,omitempty"`
}

type FontRequirement struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
}

type RuntimeSpec struct {
	Type    string   `json:"type"`
	Image   string   `json:"image,omitempty"`
	Command []string `json:"command,omitempty"`
}

type Capabilities struct {
	OutputInfo bool `json:"outputInfo,omitempty"`
}

type Manifest struct {
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"displayName"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	Author            string                 `json:"author"`
	License           string                 `json:"license"`
	MinPrestoVersion  string                 `json:"minPrestoVersion"`
	Keywords          []string               `json:"keywords"`
	RequiredFonts     []FontRequirement      `json:"requiredFonts"`
	FrontmatterSchema map[string]FieldSchema `json:"frontmatterSchema"`
	Capabilities      Capabilities           `json:"capabilities,omitempty"`
	Runtimes          []RuntimeSpec          `json:"runtimes,omitempty"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
