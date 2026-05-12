package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPreviewUpdateRejectsRelativeWorkDir(t *testing.T) {
	handler := NewServer(ServerOptions{})

	req := httptest.NewRequest(http.MethodPost, "/api/preview/update", strings.NewReader(`{
		"markdown": "# Test",
		"templateId": "missing",
		"workDir": "relative/path"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for relative workDir, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPreviewUpdateRequiresAuth(t *testing.T) {
	handler := NewServer(ServerOptions{APIKey: "secret-key"})

	req := httptest.NewRequest(http.MethodPost, "/api/preview/update", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API auth, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPreviewUpdateInvalidJSON(t *testing.T) {
	handler := NewServer(ServerOptions{})

	req := httptest.NewRequest(http.MethodPost, "/api/preview/update", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPreviewEventsDeferredBoundary(t *testing.T) {
	handler := NewServer(ServerOptions{})

	req := httptest.NewRequest(http.MethodGet, "/api/preview/events", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 for deferred preview events, got %d: %s", rec.Code, rec.Body.String())
	}

	source, err := os.ReadFile("preview.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "server Tinymist embedded renderer deferred; this endpoint preserves WebSocket/origin boundary") {
		t.Fatalf("deferred boundary comment missing from preview.go")
	}
}
