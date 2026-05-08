package api

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewServerDoesNotInjectAPIKeyWhenDisabled(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><head></head><body>ok</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}

	handler := NewServer(ServerOptions{
		StaticDir:    staticDir,
		APIKey:       "secret-key",
		InjectAPIKey: false,
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "secret-key") || strings.Contains(body, `meta name="api-key"`) {
		t.Fatalf("API key was injected when InjectAPIKey=false: %s", body)
	}
}

func TestNewServerInjectsAPIKeyWhenEnabled(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><head></head><body>ok</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}

	handler := NewServer(ServerOptions{
		StaticDir:    staticDir,
		APIKey:       "secret-key",
		InjectAPIKey: true,
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `meta name="api-key" content="secret-key"`) {
		t.Fatalf("API key meta tag was not injected: %s", body)
	}
}
