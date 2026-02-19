package template

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		manifestPath := filepath.Join(tplDir, "manifest.json")

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		manifest, err := ParseManifest(data)
		if err != nil {
			continue
		}

		binaryName := fmt.Sprintf("presto-template-%s", manifest.Name)
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(tplDir, binaryName)

		if _, err := os.Stat(binaryPath); err != nil {
			continue
		}

		templates = append(templates, InstalledTemplate{
			Manifest:   manifest,
			BinaryPath: binaryPath,
			Dir:        tplDir,
		})
	}

	// Auto-deduplicate: if multiple templates share the same manifest name,
	// rename duplicates on disk to avoid conflicts.
	seen := make(map[string]bool)
	for i := range templates {
		name := templates[i].Manifest.Name
		if !seen[name] {
			seen[name] = true
			continue
		}
		newName := uniqueNameInSet(name, seen)
		log.Printf("[templates] auto-renaming duplicate %q → %q", name, newName)
		if err := renameDiskTemplate(m.TemplatesDir, &templates[i], newName); err != nil {
			log.Printf("[templates] auto-rename failed: %v", err)
			continue
		}
		seen[newName] = true
	}

	return templates, nil
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

// Exists checks if a template with the given name is installed.
func (m *Manager) Exists(name string) bool {
	tplDir := filepath.Join(m.TemplatesDir, name)
	manifestPath := filepath.Join(tplDir, "manifest.json")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// UniqueTemplateName returns a unique name by appending -2, -3, etc. if needed.
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

// UpdateDisplayName updates the displayName field in a template's manifest.json.
func (m *Manager) UpdateDisplayName(name, newDisplayName string) error {
	name = filepath.Base(name)
	if IsOfficial(name) {
		return fmt.Errorf("cannot modify built-in template")
	}
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
	return os.WriteFile(manifestPath, out, 0644)
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
	if IsOfficial(oldName) {
		return fmt.Errorf("cannot rename built-in template")
	}
	if IsOfficial(newName) {
		return fmt.Errorf("cannot use built-in template name")
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
	oldName := t.Manifest.Name

	oldBinaryName := fmt.Sprintf("presto-template-%s", oldName)
	newBinaryName := fmt.Sprintf("presto-template-%s", newName)
	if runtime.GOOS == "windows" {
		oldBinaryName += ".exe"
		newBinaryName += ".exe"
	}

	// Rename binary file
	if err := os.Rename(filepath.Join(t.Dir, oldBinaryName), filepath.Join(t.Dir, newBinaryName)); err != nil {
		return fmt.Errorf("rename binary: %w", err)
	}

	// Update manifest name
	t.Manifest.Name = newName
	data, err := json.MarshalIndent(t.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(t.Dir, "manifest.json"), data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Rename directory
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

// EnsureOfficialTemplates copies bundled official templates from bundleDir
// to the user's templates directory if they are missing.
// bundleDir should contain subdirectories for each official template,
// e.g. bundleDir/gongwen/manifest.json and bundleDir/gongwen/presto-template-gongwen.
func (m *Manager) EnsureOfficialTemplates(bundleDir string) {
	if bundleDir == "" {
		return
	}
	for name := range OfficialTemplates {
		tplDir := filepath.Join(m.TemplatesDir, name)
		binaryName := fmt.Sprintf("presto-template-%s", name)
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}

		// Skip if already properly installed (both manifest and binary exist)
		manifestDst := filepath.Join(tplDir, "manifest.json")
		binaryDst := filepath.Join(tplDir, binaryName)
		if _, err := os.Stat(manifestDst); err == nil {
			if _, err := os.Stat(binaryDst); err == nil {
				continue
			}
		}

		// Source paths in the bundle
		manifestSrc := filepath.Join(bundleDir, name, "manifest.json")
		binarySrc := filepath.Join(bundleDir, name, binaryName)
		if _, err := os.Stat(manifestSrc); err != nil {
			log.Printf("[templates] bundled template %s not found at %s", name, bundleDir)
			continue
		}
		if _, err := os.Stat(binarySrc); err != nil {
			log.Printf("[templates] bundled binary %s not found at %s", binaryName, bundleDir)
			continue
		}

		if err := os.MkdirAll(tplDir, 0755); err != nil {
			log.Printf("[templates] failed to create dir for %s: %v", name, err)
			continue
		}
		if err := copyFile(manifestSrc, manifestDst, 0644); err != nil {
			log.Printf("[templates] failed to copy manifest for %s: %v", name, err)
			continue
		}
		if err := copyFile(binarySrc, binaryDst, 0755); err != nil {
			log.Printf("[templates] failed to copy binary for %s: %v", name, err)
			continue
		}
		log.Printf("[templates] installed bundled template: %s", name)
	}
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
