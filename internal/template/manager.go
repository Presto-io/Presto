package template

import (
	"fmt"
	"io"
	"log"
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
