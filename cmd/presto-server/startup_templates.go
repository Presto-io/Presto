package main

import (
	"log"
	"strings"

	"github.com/mrered/presto/internal/template"
)

type officialTemplateInstaller interface {
	Exists(name string) bool
	Install(owner, repo string, opts *template.InstallOpts) error
}

type officialTemplateInstallSummary struct {
	Found     int
	Installed int
	Skipped   int
	Failed    int
}

func installOfficialTemplatesOnStartup(manager officialTemplateInstaller, registry *template.RegistryCache) officialTemplateInstallSummary {
	if registry == nil {
		log.Print("[templates] registry not configured, skipping official template startup install")
		return officialTemplateInstallSummary{}
	}

	reg := registry.Load()
	if reg == nil {
		log.Print("[templates] registry unavailable, skipping official template startup install")
		return officialTemplateInstallSummary{}
	}

	summary := installOfficialTemplatesFromRegistry(manager, reg, template.Platform())
	log.Printf("[templates] official template startup install complete: found=%d installed=%d skipped=%d failed=%d",
		summary.Found, summary.Installed, summary.Skipped, summary.Failed)
	return summary
}

func installOfficialTemplatesFromRegistry(manager officialTemplateInstaller, reg *template.Registry, platform string) officialTemplateInstallSummary {
	var summary officialTemplateInstallSummary
	if manager == nil || reg == nil {
		return summary
	}

	for _, entry := range reg.Templates {
		if entry.Trust != "official" {
			continue
		}
		summary.Found++

		if manager.Exists(entry.Name) {
			summary.Skipped++
			log.Printf("[templates] official template already installed, skipping: name=%s", entry.Name)
			continue
		}

		parts := strings.SplitN(entry.Repo, "/", 2)
		if len(parts) != 2 {
			summary.Failed++
			log.Printf("[templates] invalid official template repo, skipping: name=%s repo=%s", entry.Name, entry.Repo)
			continue
		}

		opts, ok := entry.InstallOptsForPlatform(platform)
		if !ok {
			summary.Failed++
			log.Printf("[templates] official template has no binary for platform, skipping: name=%s platform=%s", entry.Name, platform)
			continue
		}

		log.Printf("[templates] installing official template on startup: name=%s platform=%s", entry.Name, platform)
		if err := manager.Install(parts[0], parts[1], opts); err != nil {
			summary.Failed++
			log.Printf("[templates] official template startup install failed: name=%s error=%v", entry.Name, err)
			continue
		}

		summary.Installed++
		log.Printf("[templates] official template installed on startup: name=%s", entry.Name)
	}

	return summary
}
