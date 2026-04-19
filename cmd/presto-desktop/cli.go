package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mrered/presto/internal/template"
)

func downloadTemplatesAndExit() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("[Installer] Starting template download...")

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Error("[Installer] Failed to get user home directory", "error", err)
		os.Exit(1)
	}

	prestoDir := filepath.Join(home, ".presto")
	templatesDir := filepath.Join(prestoDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		logger.Error("[Installer] Failed to create templates directory", "error", err)
		os.Exit(1)
	}

	manager := template.NewManager(templatesDir)
	registry := template.NewRegistryCache(prestoDir)

	reg := registry.Load()
	if reg == nil {
		logger.Error("[Installer] Failed to load template registry")
		os.Exit(1)
	}

	var officialTemplates []template.RegistryEntry
	for _, entry := range reg.Templates {
		if entry.Trust == "official" {
			officialTemplates = append(officialTemplates, entry)
		}
	}

	if len(officialTemplates) == 0 {
		logger.Info("[Installer] No official templates found in registry")
		os.Exit(0)
	}

	logger.Info("[Installer] Found official templates to download", "count", len(officialTemplates))

	platform := runtime.GOOS + "-" + runtime.GOARCH
	var failCount int
	for _, entry := range officialTemplates {
		if manager.Exists(entry.Name) {
			logger.Info("[Installer] Template already installed, skipping", "name", entry.Name)
			continue
		}

		parts := strings.SplitN(entry.Repo, "/", 2)
		if len(parts) != 2 {
			logger.Error("[Installer] Invalid repo format", "name", entry.Name, "repo", entry.Repo)
			failCount++
			continue
		}
		owner, repo := parts[0], parts[1]

		var opts *template.InstallOpts
		if info, ok := entry.Platforms[platform]; ok && info.URL != "" {
			opts = &template.InstallOpts{
				DownloadURL:    info.URL,
				CdnURL:         info.CdnURL,
				ExpectedSHA256: info.SHA256,
				Trust:          entry.Trust,
			}
		}

		logger.Info("[Installer] Downloading template", "name", entry.Name)
		if err := manager.Install(owner, repo, opts); err != nil {
			logger.Error("[Installer] Failed to download template", "name", entry.Name, "error", err)
			failCount++
			continue
		}
		logger.Info("[Installer] Successfully downloaded template", "name", entry.Name)
	}

	if failCount > 0 {
		logger.Error("[Installer] Template download completed with failures", "fail_count", failCount)
		os.Exit(1)
	}

	logger.Info("[Installer] Template download completed successfully")
	os.Exit(0)
}
