package skill

// InstalledSkill represents a skill installed in one of the AI tool skill directories.
type InstalledSkill struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Source      string   `json:"source"`      // which AI tool directory, e.g. "codex", "claude"
	Keywords    []string `json:"keywords"`
}

// ScanDir represents a source directory to scan for skills.
type ScanDir struct {
	Name string // human-readable source name, e.g. "codex"
	Path string // full directory path, e.g. "/home/user/.codex/skills/"
}
