package main

import "github.com/mrered/presto/internal/api"

var releaseChannel = "slim"

type ReleaseCapabilities struct {
	ReleaseChannel       string `json:"releaseChannel"`
	OnlineRegistry       bool   `json:"onlineRegistry"`
	OnlineTemplateStore  bool   `json:"onlineTemplateStore"`
	OnlineSkillStore     bool   `json:"onlineSkillStore"`
	TemplateAutoUpdate   bool   `json:"templateAutoUpdate"`
	FirstLaunchBootstrap bool   `json:"firstLaunchBootstrap"`
	AppUpdateCheck       bool   `json:"appUpdateCheck"`
	ExternalBrowserLinks bool   `json:"externalBrowserLinks"`
	LocalTemplateImport  bool   `json:"localTemplateImport"`
	PackagedRuntimes     bool   `json:"packagedRuntimes"`
}

func currentReleaseCapabilities() ReleaseCapabilities {
	switch releaseChannel {
	case "portable":
		return ReleaseCapabilities{
			ReleaseChannel:       "portable",
			OnlineRegistry:       false,
			OnlineTemplateStore:  false,
			OnlineSkillStore:     false,
			TemplateAutoUpdate:   false,
			FirstLaunchBootstrap: false,
			AppUpdateCheck:       false,
			ExternalBrowserLinks: false,
			LocalTemplateImport:  true,
			PackagedRuntimes:     true,
		}
	default:
		return slimReleaseCapabilities()
	}
}

func slimReleaseCapabilities() ReleaseCapabilities {
	return ReleaseCapabilities{
		ReleaseChannel:       "slim",
		OnlineRegistry:       true,
		OnlineTemplateStore:  true,
		OnlineSkillStore:     true,
		TemplateAutoUpdate:   true,
		FirstLaunchBootstrap: true,
		AppUpdateCheck:       true,
		ExternalBrowserLinks: true,
		LocalTemplateImport:  true,
		PackagedRuntimes:     false,
	}
}

func normalizeReleaseCapabilities(capabilities ReleaseCapabilities) ReleaseCapabilities {
	if capabilities.ReleaseChannel == "" {
		return currentReleaseCapabilities()
	}
	return capabilities
}

func toAPIReleaseCapabilities(capabilities ReleaseCapabilities) api.ReleaseCapabilities {
	return api.ReleaseCapabilities{
		ReleaseChannel:       capabilities.ReleaseChannel,
		OnlineRegistry:       capabilities.OnlineRegistry,
		OnlineTemplateStore:  capabilities.OnlineTemplateStore,
		OnlineSkillStore:     capabilities.OnlineSkillStore,
		TemplateAutoUpdate:   capabilities.TemplateAutoUpdate,
		FirstLaunchBootstrap: capabilities.FirstLaunchBootstrap,
		AppUpdateCheck:       capabilities.AppUpdateCheck,
		ExternalBrowserLinks: capabilities.ExternalBrowserLinks,
		LocalTemplateImport:  capabilities.LocalTemplateImport,
		PackagedRuntimes:     capabilities.PackagedRuntimes,
	}
}
