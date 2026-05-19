package main

import "testing"

func withReleaseChannel(t *testing.T, channel string) {
	t.Helper()
	previous := releaseChannel
	releaseChannel = channel
	t.Cleanup(func() {
		releaseChannel = previous
	})
}

func TestDefaultEmptyReleaseCapabilitiesUseSlim(t *testing.T) {
	withReleaseChannel(t, "")

	capabilities := currentReleaseCapabilities()
	if capabilities.ReleaseChannel != "slim" {
		t.Fatalf("expected empty release channel to normalize to slim, got %q", capabilities.ReleaseChannel)
	}
	if !capabilities.OnlineRegistry {
		t.Fatal("expected slim channel to keep online registry enabled")
	}
}

func TestPortableReleaseCapabilitiesDisableOnlineFeatures(t *testing.T) {
	withReleaseChannel(t, "portable")

	capabilities := currentReleaseCapabilities()
	if capabilities.ReleaseChannel != "portable" {
		t.Fatalf("expected portable release channel, got %q", capabilities.ReleaseChannel)
	}
	if capabilities.OnlineRegistry ||
		capabilities.OnlineTemplateStore ||
		capabilities.OnlineSkillStore ||
		capabilities.TemplateAutoUpdate ||
		capabilities.FirstLaunchBootstrap ||
		capabilities.AppUpdateCheck ||
		capabilities.ExternalBrowserLinks {
		t.Fatalf("portable capabilities must disable every online feature: %+v", capabilities)
	}
	if !capabilities.LocalTemplateImport {
		t.Fatal("portable channel must keep local template import enabled")
	}
}

func TestUnknownReleaseChannelUsesSlimDefaults(t *testing.T) {
	withReleaseChannel(t, "unexpected")

	capabilities := currentReleaseCapabilities()
	if capabilities.ReleaseChannel != "slim" {
		t.Fatalf("expected unknown release channel to normalize to slim, got %q", capabilities.ReleaseChannel)
	}
	if !capabilities.OnlineRegistry || !capabilities.OnlineTemplateStore || !capabilities.AppUpdateCheck {
		t.Fatalf("expected slim defaults for unknown channel, got %+v", capabilities)
	}
}
