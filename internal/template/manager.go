package template

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type InstalledTemplate struct {
	Manifest   *Manifest
	BinaryPath string
	Dir        string
}

type Manager struct {
	TemplatesDir string
}

func NewManager(templatesDir string) *Manager {
	return &Manager{TemplatesDir: templatesDir}
}

func templateBinaryName(name string) string {
	return templateBinaryNameForOS(name, runtime.GOOS)
}

func templateBinaryNameForOS(name string, goos string) string {
	bin := "presto-template-" + name
	if goos == "windows" {
		bin += ".exe"
	}
	return bin
}

func installArtifactLayout(goos string, manifestName string, downloadedFilename string) (binaryName string, writeManifest bool, err error) {
	if goos != "windows" {
		return templateBinaryNameForOS(manifestName, goos), true, nil
	}

	if downloadedFilename == "" {
		return "", false, fmt.Errorf("downloaded filename is required on Windows")
	}
	if strings.ContainsAny(downloadedFilename, `/\`) {
		return "", false, fmt.Errorf("downloaded filename must not contain path separators: %q", downloadedFilename)
	}
	if !strings.HasSuffix(strings.ToLower(downloadedFilename), ".exe") {
		return "", false, fmt.Errorf("downloaded Windows template must be an .exe: %q", downloadedFilename)
	}
	if err := validateName(downloadedFilename); err != nil {
		return "", false, fmt.Errorf("invalid downloaded filename: %w", err)
	}

	return downloadedFilename, false, nil
}

func (m *Manager) List() ([]InstalledTemplate, error) {
	entries, err := os.ReadDir(m.TemplatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var templates []InstalledTemplate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tplDir := filepath.Join(m.TemplatesDir, entry.Name())
		tpl, err := loadTemplateFromDir(tplDir)
		if err != nil {
			slog.Debug("[templates] skipping invalid template",
				"dir", tplDir,
				"error", err.Error())
			continue
		}
		templates = append(templates, tpl)
	}

	// Keep names stable. If duplicates exist, keep the first discovered template
	// and skip later entries instead of renaming files on disk.
	seen := make(map[string]bool)
	deduped := templates[:0]
	for _, tpl := range templates {
		name := tpl.Manifest.Name
		if seen[name] {
			slog.Warn("[templates] duplicate template skipped",
				"name", name,
				"dir", tpl.Dir)
			continue
		}
		seen[name] = true
		deduped = append(deduped, tpl)
	}

	return deduped, nil
}

func loadTemplateFromDir(tplDir string) (InstalledTemplate, error) {
	manifestPath := filepath.Join(tplDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err == nil {
		manifest, err := ParseManifest(data)
		if err != nil {
			return InstalledTemplate{}, fmt.Errorf("parse manifest: %w", err)
		}

		binaryPath := filepath.Join(tplDir, templateBinaryName(manifest.Name))
		if _, err := os.Stat(binaryPath); err != nil {
			binaryPath, err = findTemplateBinary(tplDir)
			if err != nil {
				return InstalledTemplate{}, err
			}
		}

		return InstalledTemplate{
			Manifest:   manifest,
			BinaryPath: binaryPath,
			Dir:        tplDir,
		}, nil
	}
	if !os.IsNotExist(err) {
		return InstalledTemplate{}, fmt.Errorf("read manifest: %w", err)
	}

	return loadTemplateFromBinary(tplDir)
}

func loadTemplateFromBinary(tplDir string) (InstalledTemplate, error) {
	binaryPath, err := findTemplateBinary(tplDir)
	if err != nil {
		return InstalledTemplate{}, err
	}

	manifestBytes, err := NewExecutor(binaryPath).GetManifest()
	if err != nil {
		return InstalledTemplate{}, fmt.Errorf("get manifest from binary: %w", err)
	}

	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return InstalledTemplate{}, fmt.Errorf("parse manifest from binary: %w", err)
	}
	if err := validateName(manifest.Name); err != nil {
		return InstalledTemplate{}, fmt.Errorf("invalid template name from binary: %w", err)
	}

	return InstalledTemplate{
		Manifest:   manifest,
		BinaryPath: binaryPath,
		Dir:        tplDir,
	}, nil
}

func findTemplateBinary(tplDir string) (string, error) {
	entries, err := os.ReadDir(tplDir)
	if err != nil {
		return "", fmt.Errorf("read template dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "presto-template-") {
			continue
		}
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
			continue
		}

		path := filepath.Join(tplDir, name)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		return path, nil
	}

	return "", fmt.Errorf("template binary not found")
}

func (m *Manager) Get(name string) (*InstalledTemplate, error) {
	templates, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		if t.Manifest.Name == name {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}

func (m *Manager) Executor(t *InstalledTemplate) *Executor {
	return NewExecutor(t.BinaryPath)
}

func (m *Manager) Exists(name string) bool {
	name = filepath.Base(name)
	if err := validateName(name); err != nil {
		return false
	}
	tplDir := filepath.Join(m.TemplatesDir, name)
	manifestPath := filepath.Join(tplDir, "manifest.json")
	_, err := os.Stat(manifestPath)
	if err == nil {
		return true
	}

	tpl, err := loadTemplateFromDir(tplDir)
	return err == nil && tpl.Manifest.Name == name
}

func (m *Manager) UniqueTemplateName(name string) string {
	if !m.Exists(name) {
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", name, i)
		if !m.Exists(candidate) {
			return candidate
		}
	}
}

func (m *Manager) UpdateDisplayName(name, newDisplayName string) error {
	name = filepath.Base(name)
	tplDir := filepath.Join(m.TemplatesDir, name)
	manifestPath := filepath.Join(tplDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("template %q not found", name)
	}
	manifest, err := ParseManifest(data)
	if err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}
	manifest.DisplayName = newDisplayName
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(manifestPath, out, 0600) // SEC-45
}

// Rename renames a user-installed template (directory, manifest, binary).
// SEC-06: Validates names and verifies paths stay within TemplatesDir.
func (m *Manager) Rename(oldName, newName string) error {
	oldName = filepath.Base(oldName)
	newName = filepath.Base(newName)

	if err := validateName(oldName); err != nil {
		return fmt.Errorf("invalid old name: %w", err)
	}
	if err := validateName(newName); err != nil {
		return fmt.Errorf("invalid new name: %w", err)
	}
	if oldName == newName {
		return nil
	}

	// SEC-06: Verify paths within TemplatesDir
	oldDir := filepath.Join(m.TemplatesDir, oldName)
	newDir := filepath.Join(m.TemplatesDir, newName)
	absTemplatesDir, _ := filepath.Abs(m.TemplatesDir)
	absOldDir, _ := filepath.Abs(oldDir)
	absNewDir, _ := filepath.Abs(newDir)
	if !strings.HasPrefix(absOldDir, absTemplatesDir+string(filepath.Separator)) ||
		!strings.HasPrefix(absNewDir, absTemplatesDir+string(filepath.Separator)) {
		return fmt.Errorf("path escapes templates directory")
	}

	if _, err := os.Stat(oldDir); err != nil {
		return fmt.Errorf("template %q not found", oldName)
	}
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("template %q already exists", newName)
	}

	tpl, err := m.Get(oldName)
	if err != nil {
		return err
	}

	return renameDiskTemplate(m.TemplatesDir, tpl, newName)
}

// renameDiskTemplate renames a template's binary, manifest, and directory on disk.
func renameDiskTemplate(templatesDir string, t *InstalledTemplate, newName string) error {
	oldBinaryName := templateBinaryName(t.Manifest.Name)
	newBinaryName := templateBinaryName(newName)

	if err := os.Rename(filepath.Join(t.Dir, oldBinaryName), filepath.Join(t.Dir, newBinaryName)); err != nil {
		return fmt.Errorf("rename binary: %w", err)
	}

	t.Manifest.Name = newName
	data, err := json.MarshalIndent(t.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(t.Dir, "manifest.json"), data, 0600); err != nil { // SEC-45
		return fmt.Errorf("write manifest: %w", err)
	}

	newDir := filepath.Join(templatesDir, newName)
	if err := os.Rename(t.Dir, newDir); err != nil {
		return fmt.Errorf("rename dir: %w", err)
	}

	t.BinaryPath = filepath.Join(newDir, newBinaryName)
	t.Dir = newDir
	return nil
}

func uniqueNameInSet(base string, used map[string]bool) string {
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}
