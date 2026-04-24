package skill

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillManager scans multiple AI tool skill directories for installed skills.
type SkillManager struct {
	scanDirs []ScanDir
}

// NewManager creates a SkillManager that scans the 4 known AI tool skill directories.
func NewManager() *SkillManager {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("[skills] failed to get home dir", "error", err)
		return &SkillManager{}
	}

	return &SkillManager{
		scanDirs: []ScanDir{
			{Name: "codex", Path: filepath.Join(home, ".codex", "skills")},
			{Name: "claude", Path: filepath.Join(home, ".claude", "skills")},
			{Name: "workbuddy", Path: filepath.Join(home, ".workbuddy", "skills")},
			{Name: "qclaw", Path: filepath.Join(home, ".qclaw", "skills")},
		},
	}
}

// List scans all configured skill directories and returns installed skills.
// Directories that don't exist or aren't readable are skipped silently.
// Same skill appearing in multiple tool directories produces separate entries.
func (m *SkillManager) List() ([]InstalledSkill, error) {
	var skills []InstalledSkill

	for _, dir := range m.scanDirs {
		// Skip non-existent or unreadable directories silently
		if _, err := os.Stat(dir.Path); err != nil {
			continue
		}

		entries, err := os.ReadDir(dir.Path)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Filter: only subdirectories starting with "presto-"
			if !strings.HasPrefix(entry.Name(), "presto-") {
				continue
			}

			skillDir := filepath.Join(dir.Path, entry.Name())
			skillMdPath := filepath.Join(skillDir, "SKILL.md")

			// Check if SKILL.md exists
			if _, err := os.Stat(skillMdPath); err != nil {
				continue
			}

			// Parse frontmatter from SKILL.md
			meta, err := parseFrontmatter(skillMdPath)
			if err != nil {
				slog.Warn("[skills] failed to parse frontmatter", "path", skillMdPath, "error", err)
				continue
			}

			skills = append(skills, InstalledSkill{
				Name:        entry.Name(),
				DisplayName: meta.DisplayName,
				Description: meta.Description,
				Version:     meta.Version,
				Author:      meta.Author,
				Source:      dir.Name,
				SourcePath:  skillDir,
				Keywords:    meta.Keywords,
			})
		}
	}

	// Sort by Name then Source
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Source < skills[j].Source
	})

	return skills, nil
}

// Delete removes a skill directory after validating the path is within a known scan directory.
// This prevents path traversal attacks.
func (m *SkillManager) Delete(sourcePath string) error {
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("invalid path")
	}

	// Validate that sourcePath is within one of the known scan directories
	valid := false
	for _, dir := range m.scanDirs {
		absDir, err := filepath.Abs(dir.Path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absSourcePath, absDir+string(filepath.Separator)) {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("path escapes skill directories")
	}

	if err := os.RemoveAll(absSourcePath); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	return nil
}

// frontmatterMeta holds metadata parsed from SKILL.md YAML frontmatter.
type frontmatterMeta struct {
	DisplayName string
	Description string
	Version     string
	Author      string
	Keywords    []string
}

// parseFrontmatter reads a SKILL.md file and extracts YAML frontmatter fields.
// Frontmatter is the content between the first two "---" delimiters.
func parseFrontmatter(path string) (*frontmatterMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Find first "---"
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty file")
	}
	line := strings.TrimSpace(scanner.Text())
	if line != "---" {
		return nil, fmt.Errorf("expected frontmatter delimiter, got: %s", line)
	}

	// Read lines until second "---"
	meta := &frontmatterMeta{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}

		key, value, ok := parseYamlLine(line)
		if !ok {
			continue
		}

		switch key {
		case "name", "display_name":
			if key == "display_name" && meta.DisplayName == "" {
				meta.DisplayName = value
			}
			// "name" doesn't override display_name
		case "displayName":
			meta.DisplayName = value
		case "description":
			meta.Description = value
		case "version":
			meta.Version = value
		case "author":
			meta.Author = value
		case "keywords":
			meta.Keywords = parseYamlArray(value)
		}
	}

	if meta.DisplayName == "" {
		// Fallback: use directory name as display name
		meta.DisplayName = filepath.Base(filepath.Dir(path))
	}

	return meta, nil
}

// parseYamlLine parses a simple "key: value" YAML line.
func parseYamlLine(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	// Strip surrounding quotes
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		value = value[1 : len(value)-1]
	}
	return key, value, true
}

// parseYamlArray parses a YAML inline array like "[a, b, c]".
func parseYamlArray(value string) []string {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil
	}
	inner := value[1 : len(value)-1]
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"'")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ScanDirs returns the list of configured scan directories.
func (m *SkillManager) ScanDirs() []ScanDir {
	return m.scanDirs
}
