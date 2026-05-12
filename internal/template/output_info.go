package template

import "encoding/json"

type DocumentInfo struct {
	Title       string   `json:"title,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Date        string   `json:"date,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language,omitempty"`
}

type OutputInfo struct {
	SchemaVersion  int            `json:"schemaVersion"`
	OutputBaseName string         `json:"outputBaseName"`
	PreviewTitle   string         `json:"previewTitle,omitempty"`
	Document       DocumentInfo   `json:"document,omitempty"`
	TemplateData   map[string]any `json:"templateData,omitempty"`
}

func DefaultOutputInfo() OutputInfo {
	return OutputInfo{
		SchemaVersion:  1,
		OutputBaseName: "output",
	}
}

func ParseOutputInfo(data []byte) (OutputInfo, error) {
	info := DefaultOutputInfo()
	if err := json.Unmarshal(data, &info); err != nil {
		return DefaultOutputInfo(), err
	}
	if info.SchemaVersion == 0 {
		info.SchemaVersion = 1
	}
	if info.OutputBaseName == "" {
		info.OutputBaseName = "output"
	}
	return info, nil
}
