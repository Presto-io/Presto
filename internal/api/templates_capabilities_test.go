package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func portableTestCapabilities() ReleaseCapabilities {
	capabilities := defaultReleaseCapabilities()
	capabilities.ReleaseChannel = "portable"
	capabilities.OnlineRegistry = false
	capabilities.OnlineTemplateStore = false
	capabilities.OnlineSkillStore = false
	capabilities.TemplateAutoUpdate = false
	capabilities.FirstLaunchBootstrap = false
	capabilities.AppUpdateCheck = false
	capabilities.ExternalBrowserLinks = false
	capabilities.PackagedRuntimes = true
	return capabilities
}

func TestPortableCapabilitiesDisableTemplateDiscovery(t *testing.T) {
	handler := NewServer(ServerOptions{
		TemplatesDir: t.TempDir(),
		Capabilities: portableTestCapabilities(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/templates/discover", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "online template store disabled") {
		t.Fatalf("expected disabled-store response, got %s", rec.Body.String())
	}
}

func TestPortableCapabilitiesDisableOnlineTemplateInstall(t *testing.T) {
	handler := NewServer(ServerOptions{
		TemplatesDir: t.TempDir(),
		Capabilities: portableTestCapabilities(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/templates/example/install", strings.NewReader(`{"owner":"owner","repo":"repo"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "online template store disabled") {
		t.Fatalf("expected disabled-store response, got %s", rec.Body.String())
	}
}
