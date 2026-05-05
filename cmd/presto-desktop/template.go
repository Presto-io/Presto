package main

import (
	"fmt"
	"strings"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mrered/presto/internal/template"
)

func (a *App) checkFirstLaunch() {
	logger.Debug("[first-launch] starting check", "registry_available", a.registry.Load() != nil)

	templates, err := a.manager.List()
	if err != nil {
		logger.Error("[first-launch] failed to list templates", "error", err)
		return
	}

	if len(templates) == 0 {
		logger.Info("[first-launch] first launch detected, starting default template download")
		a.downloadDefaultTemplates()
		return
	}

	logger.Info("[first-launch] templates already installed, checking for updates", "count", len(templates))
	go a.checkTemplateUpdates(templates)
}

func (a *App) downloadDefaultTemplates() {
	reg := a.registry.Load()
	if reg == nil {
		logger.Warn("[first-launch] registry not available, skipping default download")
		a.emitFirstLaunchError("无法获取模板列表")
		return
	}

	type officialTemplate struct {
		name              string
		manualDownloadURL string
	}

	var officialTemplates []officialTemplate
	for _, entry := range reg.Templates {
		if entry.Trust == "official" {
			manualDownloadURL, _ := entry.DownloadURLForPlatform(template.Platform())
			officialTemplates = append(officialTemplates, officialTemplate{
				name:              entry.Name,
				manualDownloadURL: manualDownloadURL,
			})
		}
	}

	if len(officialTemplates) == 0 {
		logger.Warn("[first-launch] no official templates found")
		return
	}

	logger.Info("[first-launch] downloading official templates", "count", len(officialTemplates))

	templateNames := make([]string, 0, len(officialTemplates))
	for _, tpl := range officialTemplates {
		templateNames = append(templateNames, tpl.name)
	}
	a.emitFirstLaunchStart(templateNames)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	var successCount int
	var failureCount int
	var mu sync.Mutex

	for _, tpl := range officialTemplates {
		wg.Add(1)
		go func(templateName string, manualDownloadURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Debug("[first-launch] downloading template", "name", templateName)
			err := a.InstallTemplate(templateName)

			mu.Lock()
			if err != nil {
				failureCount++
				logger.Error("[first-launch] failed to download template", "name", templateName, "error", err)
				a.emitFirstLaunchProgress(templateName, "error", err.Error(), manualDownloadURL)
			} else {
				successCount++
				logger.Info("[first-launch] successfully downloaded template", "name", templateName)
				a.emitFirstLaunchProgress(templateName, "success", "", "")
			}
			mu.Unlock()
		}(tpl.name, tpl.manualDownloadURL)
	}

	wg.Wait()

	a.emitFirstLaunchComplete(successCount, failureCount)
	logger.Info("[first-launch] download complete", "success", successCount, "failed", failureCount)
}

func (a *App) emitFirstLaunchStart(templateNames []string) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:start", map[string]interface{}{
		"total":     len(templateNames),
		"templates": templateNames,
	})
}

func (a *App) emitFirstLaunchProgress(name string, status string, errorMsg string, manualDownloadURL string) {
	payload := map[string]interface{}{
		"name":   name,
		"status": status,
		"error":  errorMsg,
	}
	if manualDownloadURL != "" {
		payload["manualDownloadUrl"] = manualDownloadURL
	}
	wailsRuntime.EventsEmit(a.ctx, "first-launch:progress", payload)
}

func (a *App) emitFirstLaunchComplete(success int, failed int) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:complete", map[string]int{
		"success": success,
		"failed":  failed,
	})
}

func (a *App) emitFirstLaunchError(message string) {
	wailsRuntime.EventsEmit(a.ctx, "first-launch:error", map[string]string{"message": message})
}

func (a *App) checkTemplateUpdates(installed []template.InstalledTemplate) {
	logger.Debug("[template-update] starting silent update check", "count", len(installed))

	reg := a.registry.Load()
	if reg == nil {
		logger.Warn("[template-update] registry not available, skipping update check")
		return
	}

	var updatesAvailable []struct {
		name    string
		latest  template.RegistryEntry
		current string
	}

	for _, inst := range installed {
		entry := a.registry.LookupByName(inst.Manifest.Name)
		if entry == nil {
			continue
		}

		if entry.Version != inst.Manifest.Version {
			logger.Info("[template-update] update available",
				"name", inst.Manifest.Name,
				"installed", inst.Manifest.Version,
				"latest", entry.Version)
			updatesAvailable = append(updatesAvailable, struct {
				name    string
				latest  template.RegistryEntry
				current string
			}{inst.Manifest.Name, *entry, inst.Manifest.Version})
		}
	}

	if len(updatesAvailable) == 0 {
		logger.Info("[template-update] all templates are up to date")
		return
	}

	logger.Info("[template-update] silently updating templates in background", "count", len(updatesAvailable))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	var successCount, failCount int
	var mu sync.Mutex

	for _, update := range updatesAvailable {
		wg.Add(1)
		go func(name string, entry template.RegistryEntry, oldVersion string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Info("[template-update] updating template",
				"name", name,
				"old_version", oldVersion,
				"new_version", entry.Version)

			if err := a.manager.Uninstall(name); err != nil {
				logger.Error("[template-update] failed to uninstall old version",
					"name", name,
					"error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			if err := a.InstallTemplate(name); err != nil {
				logger.Error("[template-update] failed to update template",
					"name", name,
					"error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			logger.Info("[template-update] successfully updated template",
				"name", name,
				"version", entry.Version)
			mu.Lock()
			successCount++
			mu.Unlock()
		}(update.name, update.latest, update.current)
	}

	wg.Wait()

	logger.Info("[template-update] silent update complete", "success", successCount, "failed", failCount)

	if successCount > 0 {
		wailsRuntime.EventsEmit(a.ctx, "templates-changed")
	}
}

func (a *App) InstallTemplate(templateName string) error {
	if a.registry == nil {
		return fmt.Errorf("registry not available")
	}

	entry := a.registry.LookupByName(templateName)
	if entry == nil {
		return fmt.Errorf("template %q not found in registry", templateName)
	}

	parts := strings.SplitN(entry.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", entry.Repo)
	}
	owner, repo := parts[0], parts[1]

	platform := template.Platform()
	var opts *template.InstallOpts
	if platformOpts, ok := entry.InstallOptsForPlatform(platform); ok {
		opts = platformOpts
		opts.OnProgress = func(downloaded, total int64) {
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				wailsRuntime.EventsEmit(a.ctx, "template-download:progress", map[string]interface{}{
					"template":   templateName,
					"downloaded": downloaded,
					"total":      total,
					"percent":    percent,
				})
			}
		}
		logger.Info("[templates] Wails install", "name", templateName, "trust", entry.Trust, "platform", platform)
	}

	err := a.manager.Install(owner, repo, opts)
	if err != nil {
		return err
	}

	wailsRuntime.EventsEmit(a.ctx, "templates-changed")
	wailsRuntime.EventsEmit(a.ctx, "app:notification", map[string]string{
		"message": fmt.Sprintf("模板 %s 下载完成", templateName),
		"type":    "success",
	})
	return nil
}

func (a *App) DeleteTemplate(name string) error {
	return a.manager.Uninstall(name)
}

func (a *App) GetInstalledTemplates() ([]string, error) {
	templates, err := a.manager.List()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Manifest.Name
	}
	return names, nil
}
