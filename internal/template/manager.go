package template

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
