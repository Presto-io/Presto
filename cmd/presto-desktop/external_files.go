package main

import (
	"path/filepath"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func isSupportedExternalPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".zip":
		return true
	default:
		return false
	}
}

func filterExternalPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var filtered []string
	for _, path := range paths {
		if strings.HasPrefix(path, "presto://") || !isSupportedExternalPath(path) {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		filtered = append(filtered, path)
	}
	return filtered
}

func (a *App) markFrontendReady() {
	a.externalFilesMu.Lock()
	a.frontendReady = true
	a.externalFilesMu.Unlock()
}

func (a *App) GetStartupFiles() []OpenFilesItem {
	a.externalFilesMu.Lock()
	defer a.externalFilesMu.Unlock()

	if len(a.startupFiles) == 0 {
		return nil
	}

	items := append([]OpenFilesItem(nil), a.startupFiles...)
	a.startupFiles = nil
	return items
}

func (a *App) dispatchOrQueueExternalFiles(paths []string) {
	filtered := filterExternalPaths(paths)
	if len(filtered) == 0 {
		return
	}

	items := a.readFilePaths(filtered)
	if len(items) == 0 {
		return
	}

	a.externalFilesMu.Lock()
	if !a.frontendReady || a.ctx == nil {
		a.queueStartupFilesLocked(items)
		a.externalFilesMu.Unlock()
		return
	}
	ctx := a.ctx
	a.externalFilesMu.Unlock()

	wailsRuntime.EventsEmit(ctx, "native-file-open", items)
}

func (a *App) queueStartupFilesLocked(items []OpenFilesItem) {
	seen := make(map[string]struct{}, len(a.startupFiles))
	for _, item := range a.startupFiles {
		if item.Path == "" {
			continue
		}
		seen[item.Path] = struct{}{}
	}
	for _, item := range items {
		if item.Path != "" {
			if _, ok := seen[item.Path]; ok {
				continue
			}
			seen[item.Path] = struct{}{}
		}
		a.startupFiles = append(a.startupFiles, item)
	}
}
