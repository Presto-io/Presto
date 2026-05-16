package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/mrered/presto/internal/appdata"
	"github.com/mrered/presto/internal/template"
)

func migrateLegacyDataAndExit(dirs appdata.Dirs) {
	result, err := appdata.MigrateLegacyOnce(dirs)
	if err != nil {
		logger.Warn("[Installer] Legacy data migration failed", "error", err)
		os.Exit(0)
	}
	logger.Info("[Installer] Legacy data migration finished",
		"attempted", result.Attempted,
		"skipped", result.Skipped,
		"migrated", strings.Join(result.Migrated, ","),
		"conflicts", strings.Join(result.Conflicts, ","),
		"message", result.Message)
	os.Exit(0)
}

func downloadTemplatesAndExit(dirs appdata.Dirs) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("[Installer] Starting template download...")

	if result, err := appdata.MigrateLegacyOnce(dirs); err != nil {
		logger.Warn("[Installer] Failed to migrate legacy app data", "error", err)
	} else if result.Attempted {
		logger.Info("[Installer] Legacy data migration checked",
			"migrated", strings.Join(result.Migrated, ","),
			"conflicts", strings.Join(result.Conflicts, ","),
			"message", result.Message)
	}
	if err := dirs.Ensure(); err != nil {
		logger.Error("[Installer] Failed to create app data directories", "error", err)
		os.Exit(1)
	}

	prestoDir := dirs.DataDir
	templatesDir := dirs.TemplatesDir()
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		logger.Error("[Installer] Failed to create templates directory", "error", err)
		os.Exit(1)
	}
	if err := appdata.MarkGenerated(prestoDir); err != nil {
		logger.Warn("[Installer] Failed to mark generated app data", "error", err)
	}

	manager := template.NewManager(templatesDir)
	registry := template.NewRegistryCache(dirs.CacheDir)

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

	platform := template.Platform()
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
		if platformOpts, ok := entry.InstallOptsForPlatform(platform); ok {
			opts = platformOpts
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
